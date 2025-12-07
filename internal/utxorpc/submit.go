// Copyright 2025 Blink Labs Software
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utxorpc

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"math/big"
	"sort"
	"time"

	connect "connectrpc.com/connect"
	"github.com/blinklabs-io/adder/event"
	"github.com/blinklabs-io/cardano-node-api/internal/node"
	"github.com/blinklabs-io/gouroboros/ledger"
	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
	ocommon "github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/blinklabs-io/gouroboros/protocol/localstatequery"
	"github.com/blinklabs-io/plutigo/cek"
	"github.com/blinklabs-io/plutigo/data"
	"github.com/utxorpc/go-codegen/utxorpc/v1alpha/cardano"
	submit "github.com/utxorpc/go-codegen/utxorpc/v1alpha/submit"
	"github.com/utxorpc/go-codegen/utxorpc/v1alpha/submit/submitconnect"
)

// submitServiceServer implements the SubmitService API
type submitServiceServer struct {
	submitconnect.UnimplementedSubmitServiceHandler
}

// slotLengthMs is the slot length in milliseconds (1 second for post-Shelley).
// TODO: Byron era used 20-second slots. For correct conversion across eras,
// use GetEraHistory() to map slots through era boundaries with per-era slot lengths.
const slotLengthMs = 1000

// systemStartToUnixMs converts SystemStartResult to Unix milliseconds
func systemStartToUnixMs(ss *localstatequery.SystemStartResult) int64 {
	// SystemStart contains: Year, Day (day of year 1-366), Picoseconds (within day)
	// Convert to Unix timestamp in milliseconds
	year := int(ss.Year.Int64())
	dayOfYear := ss.Day
	picoseconds := ss.Picoseconds.Int64()

	// Create time for January 1st of the year
	t := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	// Add days (dayOfYear is 1-indexed, so subtract 1)
	t = t.AddDate(0, 0, dayOfYear-1)
	// Add picoseconds (convert to nanoseconds first: pico = 10^-12, nano = 10^-9)
	nanoseconds := picoseconds / 1000
	t = t.Add(time.Duration(nanoseconds))

	return t.UnixMilli()
}

// slotToPOSIXTime converts a slot number to POSIXTime (milliseconds since Unix epoch)
func slotToPOSIXTime(slot uint64, systemStartMs int64) int64 {
	// Slot values on Cardano won't exceed int64 range for centuries
	return systemStartMs + int64(slot)*slotLengthMs //nolint:gosec
}

// buildTxOutRef creates a Plutus TxOutRef from transaction ID and index
// TxOutRef = Constr 0 [TxId, Integer]
// TxId = Constr 0 [ByteString]
func buildTxOutRef(txId []byte, index uint32) data.PlutusData {
	return data.NewConstr(0,
		data.NewConstr(0, data.NewByteString(txId)), // TxId
		data.NewInteger(big.NewInt(int64(index))),   // Index
	)
}

// buildTxInInfo creates a Plutus TxInInfo from input reference and resolved output
// TxInInfo = Constr 0 [TxOutRef, TxOut]
func buildTxInInfo(
	input lcommon.TransactionInput,
	resolvedOutput lcommon.TransactionOutput,
) data.PlutusData {
	outRef := buildTxOutRef(input.Id().Bytes(), input.Index())
	txOut := resolvedOutput.ToPlutusData()
	return data.NewConstr(0, outRef, txOut)
}

// buildValue creates a Plutus Value from lovelace amount and optional multi-asset
// Value = Map CurrencySymbol (Map TokenName Integer)
// For ADA: Map "" (Map "" amount)
func buildValue(
	lovelace *big.Int,
	assets *lcommon.MultiAsset[lcommon.MultiAssetTypeOutput],
) data.PlutusData {
	// Start with ADA (empty policy ID, empty asset name)
	adaInner := data.NewMap([][2]data.PlutusData{
		{data.NewByteString([]byte{}), data.NewInteger(lovelace)},
	})

	if assets == nil {
		return data.NewMap([][2]data.PlutusData{
			{data.NewByteString([]byte{}), adaInner},
		})
	}

	// Build the full value map including native assets
	pairs := [][2]data.PlutusData{
		{data.NewByteString([]byte{}), adaInner},
	}

	for _, policyId := range assets.Policies() {
		assetPairs := [][2]data.PlutusData{}
		for _, assetName := range assets.Assets(policyId) {
			amount := assets.Asset(policyId, assetName)
			assetPairs = append(assetPairs, [2]data.PlutusData{
				data.NewByteString(assetName),
				data.NewInteger(amount),
			})
		}
		pairs = append(pairs, [2]data.PlutusData{
			data.NewByteString(policyId.Bytes()),
			data.NewMap(assetPairs),
		})
	}

	return data.NewMap(pairs)
}

// buildMintValue creates a Plutus Value from minted/burned assets
func buildMintValue(
	mint *lcommon.MultiAsset[lcommon.MultiAssetTypeMint],
) data.PlutusData {
	if mint == nil {
		return data.NewMap([][2]data.PlutusData{})
	}

	pairs := [][2]data.PlutusData{}
	for _, policyId := range mint.Policies() {
		assetPairs := [][2]data.PlutusData{}
		for _, assetName := range mint.Assets(policyId) {
			amount := mint.Asset(policyId, assetName)
			assetPairs = append(assetPairs, [2]data.PlutusData{
				data.NewByteString(assetName),
				data.NewInteger(amount), // amount is already *big.Int
			})
		}
		pairs = append(pairs, [2]data.PlutusData{
			data.NewByteString(policyId.Bytes()),
			data.NewMap(assetPairs),
		})
	}

	return data.NewMap(pairs)
}

// buildPOSIXTimeRange creates a Plutus POSIXTimeRange from validity interval
// Interval = Constr 0 [LowerBound, UpperBound]
// LowerBound/UpperBound = Constr 0 [Extended, Closure]
// Extended: NegInf = Constr 0 [], Finite = Constr 1 [time], PosInf = Constr 2 []
// Closure: True = Constr 1 [], False = Constr 0 []
func buildPOSIXTimeRange(
	validityStart uint64,
	ttl uint64,
	systemStartMs int64,
) data.PlutusData {
	// Lower bound
	var lowerExtended data.PlutusData
	if validityStart == 0 {
		lowerExtended = data.NewConstr(0) // NegInf
	} else {
		// Convert slot to POSIXTime (milliseconds)
		posixTime := slotToPOSIXTime(validityStart, systemStartMs)
		lowerExtended = data.NewConstr(1, data.NewInteger(big.NewInt(posixTime))) // Finite
	}
	lowerBound := data.NewConstr(
		0,
		lowerExtended,
		data.NewConstr(1),
	) // Closed (True)

	// Upper bound
	var upperExtended data.PlutusData
	if ttl == 0 {
		upperExtended = data.NewConstr(2) // PosInf
	} else {
		// Convert slot to POSIXTime (milliseconds)
		posixTime := slotToPOSIXTime(ttl, systemStartMs)
		upperExtended = data.NewConstr(1, data.NewInteger(big.NewInt(posixTime))) // Finite
	}
	upperBound := data.NewConstr(
		0,
		upperExtended,
		data.NewConstr(0),
	) // Open (False) - TTL uses half-open interval [lower, upper)

	return data.NewConstr(0, lowerBound, upperBound)
}

// sortInputsCanonically returns inputs sorted in canonical order (by TxId, then Index)
// This is required because Cardano redeemer indices refer to lexicographically sorted inputs
func sortInputsCanonically(
	inputs []lcommon.TransactionInput,
) []lcommon.TransactionInput {
	sorted := make([]lcommon.TransactionInput, len(inputs))
	copy(sorted, inputs)
	sort.Slice(sorted, func(i, j int) bool {
		// Compare by TxId first (as hex string for lexicographic order)
		idI := hex.EncodeToString(sorted[i].Id().Bytes())
		idJ := hex.EncodeToString(sorted[j].Id().Bytes())
		if idI != idJ {
			return idI < idJ
		}
		// Then by output index
		return sorted[i].Index() < sorted[j].Index()
	})
	return sorted
}

// buildScriptPurpose creates a Plutus ScriptPurpose
// Minting = Constr 0 [CurrencySymbol]
// Spending = Constr 1 [TxOutRef]
// Rewarding = Constr 2 [StakingCredential]
// Certifying = Constr 3 [DCert]
func buildScriptPurpose(
	tag lcommon.RedeemerTag,
	index uint32,
	tx ledger.Transaction,
) (data.PlutusData, error) {
	switch uint8(tag) {
	case 0: // Spend
		// Sort inputs canonically - redeemer indices refer to sorted inputs
		inputs := sortInputsCanonically(tx.Inputs())
		if int(index) >= len(inputs) {
			return nil, fmt.Errorf(
				"spend redeemer index %d out of range (input count %d)",
				index,
				len(inputs),
			)
		}
		input := inputs[index]
		return data.NewConstr(
			1,
			buildTxOutRef(input.Id().Bytes(), input.Index()),
		), nil
	case 1: // Mint
		mint := tx.AssetMint()
		if mint == nil {
			return nil, fmt.Errorf(
				"mint redeemer at index %d but transaction has no minted assets",
				index,
			)
		}
		policies := mint.Policies()
		// Sort policies canonically
		sortedPolicies := make([]lcommon.Blake2b224, len(policies))
		copy(sortedPolicies, policies)
		sort.Slice(sortedPolicies, func(i, j int) bool {
			return hex.EncodeToString(
				sortedPolicies[i][:],
			) < hex.EncodeToString(
				sortedPolicies[j][:],
			)
		})
		if int(index) >= len(sortedPolicies) {
			return nil, fmt.Errorf(
				"mint redeemer index %d out of range (policy count %d)",
				index,
				len(sortedPolicies),
			)
		}
		return data.NewConstr(
			0,
			data.NewByteString(sortedPolicies[index].Bytes()),
		), nil
	case 2: // Reward
		// TODO: implement reward purpose
		return data.NewConstr(2, data.NewConstr(0)), nil
	case 3: // Cert
		// TODO: implement certifying purpose
		return data.NewConstr(3, data.NewConstr(0)), nil
	default:
		return nil, fmt.Errorf("unsupported redeemer tag: %d", tag)
	}
}

// buildTxInfo creates a Plutus TxInfo from transaction data
// TxInfo = Constr 0 [inputs, refInputs, outputs, fee, mint, dcert, wdrl, validRange, signatories, redeemers, data, txId]
func buildTxInfo(
	tx ledger.Transaction,
	resolvedUtxos map[string]ledger.Utxo,
	redeemers lcommon.TransactionWitnessRedeemers,
	systemStartMs int64,
) data.PlutusData {
	// Build inputs as [TxInInfo]
	inputsList := []data.PlutusData{}
	for _, input := range tx.Inputs() {
		if utxo, ok := resolvedUtxos[input.String()]; ok {
			inputsList = append(inputsList, buildTxInInfo(input, utxo.Output))
		}
	}
	inputs := data.NewList(inputsList...)

	// Build reference inputs as [TxInInfo]
	refInputsList := []data.PlutusData{}
	for _, input := range tx.ReferenceInputs() {
		if utxo, ok := resolvedUtxos[input.String()]; ok {
			refInputsList = append(
				refInputsList,
				buildTxInInfo(input, utxo.Output),
			)
		}
	}
	refInputs := data.NewList(refInputsList...)

	// Build outputs as [TxOut]
	txOutputs := tx.Outputs()
	outputsList := make([]data.PlutusData, 0, len(txOutputs))
	for _, output := range txOutputs {
		outputsList = append(outputsList, output.ToPlutusData())
	}
	outputs := data.NewList(outputsList...)

	// Build fee as Value (ADA only)
	fee := buildValue(tx.Fee(), nil)

	// Build mint as Value
	mint := buildMintValue(tx.AssetMint())

	// Build certificates as [DCert] (empty for now, can be extended)
	dcerts := data.NewList()

	// Build withdrawals as Map StakingCredential Integer
	wdrlPairs := [][2]data.PlutusData{}
	for addr, amount := range tx.Withdrawals() {
		if addr != nil {
			wdrlPairs = append(wdrlPairs, [2]data.PlutusData{
				addr.ToPlutusData(),
				data.NewInteger(amount),
			})
		}
	}
	wdrl := data.NewMap(wdrlPairs)

	// Build validity range (convert slots to POSIXTime)
	validRange := buildPOSIXTimeRange(
		tx.ValidityIntervalStart(),
		tx.TTL(),
		systemStartMs,
	)

	// Build signatories as [PubKeyHash]
	reqSigners := tx.RequiredSigners()
	sigsList := make([]data.PlutusData, 0, len(reqSigners))
	for _, signer := range reqSigners {
		sigsList = append(sigsList, data.NewByteString(signer.Bytes()))
	}
	signatories := data.NewList(sigsList...)

	// Build redeemers map: Map ScriptPurpose Redeemer
	redeemerPairs := [][2]data.PlutusData{} //nolint:prealloc
	for key, value := range redeemers.Iter() {
		purpose, err := buildScriptPurpose(key.Tag, key.Index, tx)
		if err != nil {
			log.Printf(
				"Skipping redeemer in TxInfo map (tag=%d index=%d): %v",
				key.Tag,
				key.Index,
				err,
			)
			continue
		}
		redeemerPairs = append(redeemerPairs, [2]data.PlutusData{
			purpose,
			value.Data.Data,
		})
	}
	redeemersMap := data.NewMap(redeemerPairs)

	// Build datums map: Map DatumHash Datum
	datumPairs := [][2]data.PlutusData{}
	witnesses := tx.Witnesses()
	if witnesses != nil {
		for _, datum := range witnesses.PlutusData() {
			datumPairs = append(datumPairs, [2]data.PlutusData{
				data.NewByteString(datum.Hash().Bytes()),
				datum.Data,
			})
		}
	}
	datumsMap := data.NewMap(datumPairs)

	// Transaction ID
	txId := data.NewConstr(0, data.NewByteString(tx.Hash().Bytes()))

	return data.NewConstr(0,
		inputs,
		refInputs,
		outputs,
		fee,
		mint,
		dcerts,
		wdrl,
		validRange,
		signatories,
		redeemersMap,
		datumsMap,
		txId,
	)
}

// buildScriptContext creates a complete Plutus ScriptContext
// ScriptContext = Constr 0 [TxInfo, ScriptPurpose]
func buildScriptContext(
	tx ledger.Transaction,
	resolvedUtxos map[string]ledger.Utxo,
	redeemers lcommon.TransactionWitnessRedeemers,
	redeemerTag lcommon.RedeemerTag,
	redeemerIndex uint32,
	systemStartMs int64,
) (data.PlutusData, error) {
	txInfo := buildTxInfo(tx, resolvedUtxos, redeemers, systemStartMs)
	purpose, err := buildScriptPurpose(redeemerTag, redeemerIndex, tx)
	if err != nil {
		return nil, fmt.Errorf("build script purpose: %w", err)
	}
	return data.NewConstr(0, txInfo, purpose), nil
}

// SubmitTx
func (s *submitServiceServer) SubmitTx(
	ctx context.Context,
	req *connect.Request[submit.SubmitTxRequest],
) (*connect.Response[submit.SubmitTxResponse], error) {
	// txRaw
	txRaw := req.Msg.GetTx() // *AnyChainTx
	if txRaw == nil {
		return nil, errors.New("transaction is required")
	}
	log.Printf("Got a SubmitTx request with 1 transaction")
	resp := &submit.SubmitTxResponse{}

	// Connect to node
	oConn, err := node.GetConnection(nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		// Close Ouroboros connection
		oConn.Close()
	}()

	txRawBytes := txRaw.GetRaw() // raw bytes
	txType, err := ledger.DetermineTransactionType(txRawBytes)
	if err != nil {
		resp.Ref = []byte{}
		return connect.NewResponse(resp), err
	}
	tx, err := ledger.NewTransactionFromCbor(txType, txRawBytes)
	if err != nil {
		resp.Ref = []byte{}
		return connect.NewResponse(resp), err
	}
	// Submit the transaction
	err = oConn.LocalTxSubmission().Client.SubmitTx(
		uint16(txType), // #nosec G115
		txRawBytes,
	)
	if err != nil {
		resp.Ref = []byte{}
		return connect.NewResponse(resp), err
	}
	resp.Ref = tx.Hash().Bytes()
	return connect.NewResponse(resp), nil
}

// convertPlutusData converts plutigo data.PlutusData to *cardano.PlutusData
func convertPlutusData(pd data.PlutusData) (*cardano.PlutusData, error) {
	switch v := pd.(type) {
	case *data.Constr:
		fields := make([]*cardano.PlutusData, len(v.Fields))
		for i, field := range v.Fields {
			converted, err := convertPlutusData(field)
			if err != nil {
				return nil, err
			}
			fields[i] = converted
		}
		return &cardano.PlutusData{
			PlutusData: &cardano.PlutusData_Constr{
				Constr: &cardano.Constr{
					Tag:    uint32(v.Tag), //nolint:gosec
					Fields: fields,
				},
			},
		}, nil
	case *data.Map:
		pairs := make([]*cardano.PlutusDataPair, len(v.Pairs))
		for i, pair := range v.Pairs {
			key, err := convertPlutusData(pair[0])
			if err != nil {
				return nil, err
			}
			value, err := convertPlutusData(pair[1])
			if err != nil {
				return nil, err
			}
			pairs[i] = &cardano.PlutusDataPair{
				Key:   key,
				Value: value,
			}
		}
		return &cardano.PlutusData{
			PlutusData: &cardano.PlutusData_Map{
				Map: &cardano.PlutusDataMap{
					Pairs: pairs,
				},
			},
		}, nil
	case *data.Integer:
		// Check if the integer fits in int64
		if v.Inner.IsInt64() {
			return &cardano.PlutusData{
				PlutusData: &cardano.PlutusData_BigInt{
					BigInt: &cardano.BigInt{
						BigInt: &cardano.BigInt_Int{
							Int: v.Inner.Int64(),
						},
					},
				},
			}, nil
		}
		// For larger integers, use BigUInt for positive and BigNInt for negative
		if v.Inner.Sign() < 0 {
			// For negative big integers, get absolute value bytes
			absVal := new(big.Int).Abs(v.Inner)
			return &cardano.PlutusData{
				PlutusData: &cardano.PlutusData_BigInt{
					BigInt: &cardano.BigInt{
						BigInt: &cardano.BigInt_BigNInt{
							BigNInt: absVal.Bytes(),
						},
					},
				},
			}, nil
		}
		// Use BigUInt for larger positive integers
		return &cardano.PlutusData{
			PlutusData: &cardano.PlutusData_BigInt{
				BigInt: &cardano.BigInt{
					BigInt: &cardano.BigInt_BigUInt{
						BigUInt: v.Inner.Bytes(),
					},
				},
			},
		}, nil
	case *data.ByteString:
		return &cardano.PlutusData{
			PlutusData: &cardano.PlutusData_BoundedBytes{
				BoundedBytes: v.Inner,
			},
		}, nil
	case *data.List:
		items := make([]*cardano.PlutusData, len(v.Items))
		for i, item := range v.Items {
			converted, err := convertPlutusData(item)
			if err != nil {
				return nil, err
			}
			items[i] = converted
		}
		return &cardano.PlutusData{
			PlutusData: &cardano.PlutusData_Array{
				Array: &cardano.PlutusDataArray{
					Items: items,
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported PlutusData type: %T", pd)
	}
}

// EvalTx evaluates scripts in a transaction and returns estimated execution units.
func (s *submitServiceServer) EvalTx(
	ctx context.Context,
	req *connect.Request[submit.EvalTxRequest],
) (*connect.Response[submit.EvalTxResponse], error) {
	// txRaw
	txRaw := req.Msg.GetTx() // *AnyChainTx
	if txRaw == nil {
		return nil, errors.New("transaction is required")
	}
	log.Printf("Got an EvalTx request")

	resp := &submit.EvalTxResponse{}

	// Connect to node
	oConn, err := node.GetConnection(nil)
	if err != nil {
		return connect.NewResponse(resp), err
	}
	defer func() {
		// Close Ouroboros connection
		oConn.Close()
	}()

	// Parse the transaction
	txRawBytes := txRaw.GetRaw() // raw bytes
	txType, err := ledger.DetermineTransactionType(txRawBytes)
	if err != nil {
		return connect.NewResponse(resp), err
	}
	tx, err := ledger.NewTransactionFromCbor(txType, txRawBytes)
	if err != nil {
		return connect.NewResponse(resp), err
	}

	// Get protocol parameters for cost models
	oConn.LocalStateQuery().Client.Start()
	protoParams, err := oConn.LocalStateQuery().Client.GetCurrentProtocolParams()
	if err != nil {
		return connect.NewResponse(resp), err
	}

	// Get system start for slot-to-time conversion
	systemStart, err := oConn.LocalStateQuery().Client.GetSystemStart()
	if err != nil {
		return connect.NewResponse(
				resp,
			), fmt.Errorf(
				"get system start: %w",
				err,
			)
	}
	systemStartMs := systemStartToUnixMs(systemStart)

	// Resolve UTxOs for all inputs (regular + reference) in a single batch query
	allInputs := make(
		[]ledger.TransactionInput,
		0,
		len(tx.Inputs())+len(tx.ReferenceInputs()),
	)
	allInputs = append(allInputs, tx.Inputs()...)
	allInputs = append(allInputs, tx.ReferenceInputs()...)

	resolvedUtxos := make(map[string]ledger.Utxo)
	if len(allInputs) > 0 {
		utxos, err := oConn.LocalStateQuery().Client.GetUTxOByTxIn(
			allInputs,
		)
		if err != nil {
			return connect.NewResponse(resp), fmt.Errorf(
				"failed to query UTxOs: %w",
				err,
			)
		}
		for utxoId, output := range utxos.Results {
			// Match results back to inputs by TxHash and Index
			for _, input := range allInputs {
				if hex.EncodeToString(
					utxoId.Hash[:],
				) == hex.EncodeToString(
					input.Id().Bytes(),
				) &&
					utxoId.Idx == int(input.Index()) {
					resolvedUtxos[input.String()] = ledger.Utxo{
						Id:     input,
						Output: output,
					}
					break
				}
			}
		}
	}

	// Get witnesses
	witnesses := tx.Witnesses()
	if witnesses == nil {
		return connect.NewResponse(
				resp,
			), errors.New(
				"transaction witnesses are nil",
			)
	}
	redeemers := witnesses.Redeemers()
	if redeemers == nil {
		return connect.NewResponse(
				resp,
			), errors.New(
				"transaction redeemers are nil",
			)
	}

	// Get cost models and protocol version from protocol parameters
	var costModels map[uint][]int64
	var protoVersionMajor, protoVersionMinor uint
	switch params := protoParams.(type) {
	case *ledger.ConwayProtocolParameters:
		costModels = params.CostModels
		protoVersionMajor = params.ProtocolVersion.Major
		protoVersionMinor = params.ProtocolVersion.Minor
	case *ledger.BabbageProtocolParameters:
		costModels = params.CostModels
		protoVersionMajor = params.ProtocolMajor
		protoVersionMinor = params.ProtocolMinor
	case *ledger.AlonzoProtocolParameters:
		costModels = params.CostModels
		protoVersionMajor = params.ProtocolMajor
		protoVersionMinor = params.ProtocolMinor
	default:
		return connect.NewResponse(resp), fmt.Errorf("unsupported protocol parameters type for cost models: %T", protoParams)
	}
	protoVersion := cek.ProtoVersion{
		Major: protoVersionMajor,
		Minor: protoVersionMinor,
	}
	if costModels == nil {
		return connect.NewResponse(
				resp,
			), errors.New(
				"cost models are not available for this era",
			)
	}

	// Build maps from script hash to script object for each version
	v1ScriptByHash := make(map[string]lcommon.PlutusV1Script)
	v2ScriptByHash := make(map[string]lcommon.PlutusV2Script)
	v3ScriptByHash := make(map[string]lcommon.PlutusV3Script)

	for _, s := range witnesses.PlutusV1Scripts() {
		hash := s.Hash()
		v1ScriptByHash[hex.EncodeToString(hash[:])] = s
	}
	for _, s := range witnesses.PlutusV2Scripts() {
		hash := s.Hash()
		v2ScriptByHash[hex.EncodeToString(hash[:])] = s
	}
	for _, s := range witnesses.PlutusV3Scripts() {
		hash := s.Hash()
		v3ScriptByHash[hex.EncodeToString(hash[:])] = s
	}

	// Count redeemers for pre-allocation
	redeemerCount := 0
	for range redeemers.Iter() {
		redeemerCount++
	}

	type redeemerPair struct {
		key   lcommon.RedeemerKey
		value lcommon.RedeemerValue
	}

	pairs := make([]redeemerPair, 0, redeemerCount)
	for key, value := range redeemers.Iter() {
		pairs = append(pairs, redeemerPair{key, value})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].key.Tag != pairs[j].key.Tag {
			return pairs[i].key.Tag < pairs[j].key.Tag
		}
		return pairs[i].key.Index < pairs[j].key.Index
	})

	txEvalRedeemers := make([]*cardano.Redeemer, 0, len(pairs))
	for _, pair := range pairs {
		key := pair.key
		value := pair.value
		payload, err := convertPlutusData(value.Data.Data)
		if err != nil {
			log.Printf(
				"Failed to convert redeemer payload for index %d: %v",
				key.Index,
				err,
			)
			payload = nil // Set to nil to avoid nil pointer issues
		}
		redeemer := &cardano.Redeemer{
			Purpose: cardano.RedeemerPurpose(key.Tag),
			Payload: payload,
			Index:   key.Index,
			ExUnits: &cardano.ExUnits{
				Steps:  0,
				Memory: 0,
			},
		}

		// Look up the script based on redeemer purpose
		var scriptHashHex string
		var args []data.PlutusData

		// For spending scripts: [datum, redeemer, context]
		if uint8(key.Tag) == 0 { // RedeemerTagSpend
			// Sort inputs canonically - redeemer indices refer to sorted inputs
			inputs := sortInputsCanonically(tx.Inputs())
			if key.Index >= uint32(len(inputs)) { //nolint:gosec
				return connect.NewResponse(resp), fmt.Errorf(
					"redeemer index out of range: index %d >= input count %d",
					key.Index,
					len(inputs),
				)
			}
			input := inputs[key.Index]
			utxo, ok := resolvedUtxos[input.String()]
			if !ok {
				return connect.NewResponse(
						resp,
					), fmt.Errorf(
						"UTxO not found for input %s",
						input.String(),
					)
			}

			// Get script hash from the UTxO address (payment credential)
			addr := utxo.Output.Address()
			addrType := addr.Type()
			// Check if payment part is a script (odd type values have script payment)
			if addrType&lcommon.AddressTypeScriptBit != 0 {
				paymentHash := addr.PaymentKeyHash()
				scriptHashHex = hex.EncodeToString(paymentHash[:])
			} else {
				log.Printf(
					"Input %s has key address, not script address",
					input.String(),
				)
				txEvalRedeemers = append(txEvalRedeemers, redeemer)
				continue
			}

			datum := utxo.Output.Datum()
			if datum == nil {
				return connect.NewResponse(
						resp,
					), fmt.Errorf(
						"no datum found for input %s",
						input.String(),
					)
			}
			datumData := datum.Data
			redeemerData := value.Data.Data
			contextData, err := buildScriptContext(
				tx,
				resolvedUtxos,
				redeemers,
				key.Tag,
				key.Index,
				systemStartMs,
			)
			if err != nil {
				return connect.NewResponse(resp), fmt.Errorf(
					"build script context for spend redeemer %d: %w",
					key.Index,
					err,
				)
			}
			args = []data.PlutusData{datumData, redeemerData, contextData}
		} else if uint8(key.Tag) == 1 { // RedeemerTagMint
			// For minting, the script hash is the policy ID
			mint := tx.AssetMint()
			if mint == nil {
				log.Printf("Mint redeemer but no mint in transaction")
				txEvalRedeemers = append(txEvalRedeemers, redeemer)
				continue
			}
			policies := mint.Policies()
			if int(key.Index) >= len(policies) {
				return connect.NewResponse(resp), fmt.Errorf(
					"mint redeemer index out of range: index %d >= policy count %d",
					key.Index,
					len(policies),
				)
			}
			// Sort policies to match canonical ordering
			sortedPolicies := make([]lcommon.Blake2b224, len(policies))
			copy(sortedPolicies, policies)
			sort.Slice(sortedPolicies, func(i, j int) bool {
				return hex.EncodeToString(sortedPolicies[i][:]) < hex.EncodeToString(sortedPolicies[j][:])
			})
			policyId := sortedPolicies[key.Index]
			scriptHashHex = hex.EncodeToString(policyId[:])

			redeemerData := value.Data.Data
			contextData, err := buildScriptContext(
				tx,
				resolvedUtxos,
				redeemers,
				key.Tag,
				key.Index,
				systemStartMs,
			)
			if err != nil {
				return connect.NewResponse(resp), fmt.Errorf(
					"build script context for mint redeemer %d: %w",
					key.Index,
					err,
				)
			}
			args = []data.PlutusData{redeemerData, contextData}
		} else {
			// For rewarding/certifying: not yet implemented
			// TODO: implement proper script hash lookup for reward/cert
			log.Printf(
				"Reward/cert redeemer evaluation not yet implemented (tag=%d index=%d)",
				key.Tag,
				key.Index,
			)
			txEvalRedeemers = append(txEvalRedeemers, redeemer)
			continue
		}

		// Look up script by hash and evaluate using gouroboros methods
		var exUnits lcommon.ExUnits
		var evalErr error

		// Create a large budget for evaluation (will return actual usage)
		maxBudget := lcommon.ExUnits{Memory: 14_000_000, Steps: 10_000_000_000}

		if v1Script, found := v1ScriptByHash[scriptHashHex]; found {
			// V1 script: Evaluate(datum, redeemer, scriptContext, budget, evalContext)
			costModelData := costModels[1]
			langVersion := cek.LanguageVersionV1
			evalContext, err := cek.NewEvalContext(
				langVersion,
				protoVersion,
				costModelData,
			)
			if err != nil {
				log.Printf(
					"Failed to create eval context for V1 script: %v",
					err,
				)
				txEvalRedeemers = append(txEvalRedeemers, redeemer)
				continue
			}
			if len(args) >= 3 {
				exUnits, evalErr = v1Script.Evaluate(
					args[0],
					args[1],
					args[2],
					maxBudget,
					evalContext,
				)
			} else if len(args) >= 2 {
				// Minting script: no datum, args = [redeemer, context]
				exUnits, evalErr = v1Script.Evaluate(nil, args[0], args[1], maxBudget, evalContext)
			}
		} else if v2Script, found := v2ScriptByHash[scriptHashHex]; found {
			// V2 script: Evaluate(datum, redeemer, scriptContext, budget, evalContext)
			costModelData := costModels[2]
			langVersion := cek.LanguageVersionV2
			evalContext, err := cek.NewEvalContext(langVersion, protoVersion, costModelData)
			if err != nil {
				log.Printf("Failed to create eval context for V2 script: %v", err)
				txEvalRedeemers = append(txEvalRedeemers, redeemer)
				continue
			}
			if len(args) >= 3 {
				exUnits, evalErr = v2Script.Evaluate(args[0], args[1], args[2], maxBudget, evalContext)
			} else if len(args) >= 2 {
				// Minting script: no datum, args = [redeemer, context]
				exUnits, evalErr = v2Script.Evaluate(nil, args[0], args[1], maxBudget, evalContext)
			}
		} else if v3Script, found := v3ScriptByHash[scriptHashHex]; found {
			// V3 script: Evaluate(scriptContext, budget, evalContext)
			// For V3, datum and redeemer are embedded in the scriptContext
			costModelData := costModels[3]
			langVersion := cek.LanguageVersionV3
			evalContext, err := cek.NewEvalContext(langVersion, protoVersion, costModelData)
			if err != nil {
				log.Printf("Failed to create eval context for V3 script: %v", err)
				txEvalRedeemers = append(txEvalRedeemers, redeemer)
				continue
			}
			// V3 scriptContext should include datum and redeemer
			scriptContext := args[len(args)-1] // Last arg is always context
			exUnits, evalErr = v3Script.Evaluate(scriptContext, maxBudget, evalContext)
		} else {
			log.Printf(
				"Script not found for hash %s (redeemer tag=%d index=%d)",
				scriptHashHex,
				key.Tag,
				key.Index,
			)
			txEvalRedeemers = append(txEvalRedeemers, redeemer)
			continue
		}

		if evalErr != nil {
			log.Printf(
				"Failed to evaluate script for redeemer %d: %v",
				key.Index,
				evalErr,
			)
		} else {
			redeemer.ExUnits.Steps = uint64(exUnits.Steps)   //nolint:gosec
			redeemer.ExUnits.Memory = uint64(exUnits.Memory) //nolint:gosec
		}

		txEvalRedeemers = append(txEvalRedeemers, redeemer)
	}

	resp.Report = &submit.AnyChainEval{
		Chain: &submit.AnyChainEval_Cardano{
			Cardano: &cardano.TxEval{
				Fee: &cardano.BigInt{
					BigInt: &cardano.BigInt_Int{
						Int: 0,
					},
				},
				Redeemers: txEvalRedeemers,
			},
		},
	}

	return connect.NewResponse(resp), nil
}

func (s *submitServiceServer) WaitForTx(
	ctx context.Context,
	req *connect.Request[submit.WaitForTxRequest],
	stream *connect.ServerStream[submit.WaitForTxResponse],
) error {
	logger := slog.With("component", "WaitForTx")
	ref := req.Msg.GetRef() // [][]byte
	logger.Info("Received WaitForTx request", "transaction_count", len(ref))

	// Log the transaction references at debug level
	for i, r := range ref {
		logger.Debug(
			"Transaction reference",
			"index",
			i,
			"ref",
			hex.EncodeToString(r),
		)
	}

	// Setup event channel
	eventChan := make(
		chan event.Event,
		100,
	) // Increased buffer size for high-throughput
	connCfg := node.ConnectionConfig{
		ChainSyncEventChan: eventChan,
	}

	// Connect to node
	logger.Debug("Establishing connection to Ouroboros node...")
	oConn, err := node.GetConnection(&connCfg)
	if err != nil {
		logger.Error("Failed to connect to node", "error", err)
		return err
	}
	defer func() {
		logger.Debug("Closing connection to Ouroboros node.")
		oConn.Close()
	}()

	// Get the current chain tip
	tip, err := oConn.ChainSync().Client.GetCurrentTip()
	if err != nil {
		logger.Error("Error retrieving current tip", "error", err)
		return err
	}
	logger.Debug("Current chain tip retrieved", "tip", tip)

	// Start the sync with the node
	logger.Debug("Starting chain synchronization...")
	err = oConn.ChainSync().Client.Sync([]ocommon.Point{tip.Point})
	if err != nil {
		logger.Error("Error during chain synchronization", "error", err)
		return err
	}

	// Context cancellation handling
	go func() {
		<-ctx.Done()
		logger.Debug("Client canceled the request. Stopping event processing.")
		close(eventChan)
	}()

	// Wait for events
	logger.Debug("Waiting for transaction events...")
	for {
		select {
		case <-ctx.Done():
			logger.Info("Context canceled. Exiting event loop.")
			return ctx.Err()
		case evt, ok := <-eventChan:
			if !ok {
				logger.Error("Event channel closed unexpectedly.")
				return errors.New("event channel closed")
			}

			// Process the event
			switch v := evt.Payload.(type) {
			case event.TransactionEvent:
				logger.Debug("Received TransactionEvent", "hash", v.Transaction.Hash().String())
				for _, r := range ref {
					refHash := hex.EncodeToString(r)
					eventHash := v.Transaction.Hash().String()

					logger.Debug("Comparing TransactionEvent with reference", "event_hash", eventHash, "reference_hash", refHash)
					if refHash == eventHash {
						logger.Info("Transaction matches reference", "hash", eventHash)

						// Send confirmation response
						err = stream.Send(&submit.WaitForTxResponse{
							Ref:   r,
							Stage: submit.Stage_STAGE_CONFIRMED,
						})
						if err != nil {
							if ctx.Err() != nil {
								logger.Warn("Client disconnected while sending response", "error", ctx.Err())
								return ctx.Err()
							}
							logger.Error("Error sending response to client", "transaction_hash", eventHash, "error", err)
							return err
						}
						logger.Info("Confirmation response sent", "transaction_hash", eventHash)
						return nil // Stop processing after confirming the transaction
					}
				}
			default:
				logger.Debug("Received unsupported event type", "type", evt.Type)
			}
		}
	}
}

// ReadMempool
func (s *submitServiceServer) ReadMempool(
	ctx context.Context,
	req *connect.Request[submit.ReadMempoolRequest],
) (*connect.Response[submit.ReadMempoolResponse], error) {
	log.Printf("Got a ReadMempool request")
	resp := &submit.ReadMempoolResponse{}

	// Connect to node
	oConn, err := node.GetConnection(nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		// Close Ouroboros connection
		oConn.Close()
	}()

	// Start LocalTxMonitor client
	oConn.LocalTxMonitor().Client.Start()

	// Collect TX hashes from the mempool
	mempool := []*submit.TxInMempool{}
	for {
		txRawBytes, err := oConn.LocalTxMonitor().Client.NextTx()
		if err != nil {
			log.Printf("ERROR: %s", err)
			return nil, err
		}
		// No transactions in mempool
		if txRawBytes == nil {
			break
		}

		record := &submit.TxInMempool{
			NativeBytes: txRawBytes,
			Stage:       submit.Stage_STAGE_MEMPOOL,
		}
		mempool = append(mempool, record)
	}

	resp.Items = mempool
	return connect.NewResponse(resp), nil
}

// WatchMempool
func (s *submitServiceServer) WatchMempool(
	ctx context.Context,
	req *connect.Request[submit.WatchMempoolRequest],
	stream *connect.ServerStream[submit.WatchMempoolResponse],
) error {
	predicate := req.Msg.GetPredicate() // Predicate
	fieldMask := req.Msg.GetFieldMask()
	log.Printf(
		"Got a WatchMempool request with predicate %v and fieldMask %v",
		predicate,
		fieldMask,
	)

	// Connect to node
	oConn, err := node.GetConnection(nil)
	if err != nil {
		return err
	}
	defer func() {
		// Close Ouroboros connection
		oConn.Close()
	}()

	// Start clients
	oConn.LocalTxMonitor().Client.Start()

	// Collect TX hashes from the mempool
	needsAcquire := false
	for {
		if needsAcquire {
			err = oConn.LocalTxMonitor().Client.Acquire()
			if err != nil {
				log.Printf("ERROR: %s", err)
				return err
			}
		}
		txRawBytes, err := oConn.LocalTxMonitor().Client.NextTx()
		if err != nil {
			log.Printf("ERROR: %s", err)
			return err
		}
		// No transactions in mempool, release and continue
		if txRawBytes == nil {
			err := oConn.LocalTxMonitor().Client.Release()
			if err != nil {
				log.Printf("ERROR: %s", err)
				return err
			}
			needsAcquire = true
			continue
		}

		txType, err := ledger.DetermineTransactionType(txRawBytes)
		if err != nil {
			return err
		}
		tx, err := ledger.NewTransactionFromCbor(txType, txRawBytes)
		if err != nil {
			return err
		}
		cTx, err := tx.Utxorpc() // *cardano.Tx
		if err != nil {
			return fmt.Errorf("convert transaction: %w", err)
		}
		resp := &submit.WatchMempoolResponse{}
		record := &submit.TxInMempool{
			NativeBytes: txRawBytes,
			Stage:       submit.Stage_STAGE_MEMPOOL,
		}
		resp.Tx = record
		if string(record.GetNativeBytes()) == cTx.String() {
			if predicate == nil {
				err := stream.Send(resp)
				if err != nil {
					return err
				}
			} else {
				// TODO: filter from all Predicate types
				err := stream.Send(resp)
				if err != nil {
					return err
				}
			}
		}
	}
}
