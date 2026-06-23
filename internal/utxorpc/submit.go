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
	"math"
	"math/big"
	"sort"
	"time"

	connect "connectrpc.com/connect"
	"github.com/blinklabs-io/adder/event"
	"github.com/blinklabs-io/cardano-node-api/internal/node"
	"github.com/blinklabs-io/gouroboros/ledger"
	lcommon "github.com/blinklabs-io/gouroboros/ledger/common"
	script "github.com/blinklabs-io/gouroboros/ledger/common/script"
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

func slotDurationMs(slotCount uint64, slotLengthMs int64) (int64, error) {
	if slotCount > uint64(math.MaxInt64) {
		return 0, fmt.Errorf("slot span %d exceeds int64 range", slotCount)
	}
	slotCountInt := int64(
		slotCount,
	) //nolint:gosec // checked against MaxInt64 above
	if slotCountInt > math.MaxInt64/slotLengthMs {
		return 0, fmt.Errorf(
			"slot span %d overflows milliseconds with slot length %d",
			slotCount,
			slotLengthMs,
		)
	}
	return slotCountInt * slotLengthMs, nil
}

func addMilliseconds(base int64, delta int64) (int64, error) {
	if delta > 0 && base > math.MaxInt64-delta {
		return 0, errors.New("POSIX time exceeds int64 range")
	}
	if delta < 0 && base < math.MinInt64-delta {
		return 0, errors.New("POSIX time is below int64 range")
	}
	return base + delta, nil
}

// slotToPOSIXTime converts a slot number to POSIXTime (milliseconds since Unix epoch)
func slotToPOSIXTime(
	slot uint64,
	systemStartMs int64,
	eraHistory []localstatequery.EraHistoryResult,
) (int64, error) {
	if len(eraHistory) == 0 {
		return 0, errors.New("era history is empty")
	}

	eras := make([]localstatequery.EraHistoryResult, len(eraHistory))
	copy(eras, eraHistory)
	sort.Slice(eras, func(i, j int) bool {
		return eras[i].Begin.SlotNo < eras[j].Begin.SlotNo
	})

	var elapsedMs int64
	for _, era := range eras {
		if era.Begin.SlotNo < 0 || era.End.SlotNo < 0 {
			return 0, fmt.Errorf(
				"era history contains negative slot boundary: begin=%d end=%d",
				era.Begin.SlotNo,
				era.End.SlotNo,
			)
		}
		if era.Params.SlotLength <= 0 {
			return 0, fmt.Errorf(
				"era history contains invalid slot length %d",
				era.Params.SlotLength,
			)
		}

		beginSlot := uint64(era.Begin.SlotNo) //nolint:gosec
		endSlot := uint64(era.End.SlotNo)     //nolint:gosec
		if endSlot <= beginSlot {
			endSlot = ^uint64(0)
		}
		// The hard-fork era history reports SlotLength already in
		// milliseconds (Ouroboros encodes it via slotLengthToMillisec), so it
		// must not be scaled by 1000 again.
		slotLengthMs := int64(era.Params.SlotLength)

		if slot < beginSlot {
			return 0, fmt.Errorf(
				"slot %d is before first era boundary %d",
				slot,
				beginSlot,
			)
		}
		if slot < endSlot {
			deltaMs, err := slotDurationMs(slot-beginSlot, slotLengthMs)
			if err != nil {
				return 0, err
			}
			posixTime, err := addMilliseconds(systemStartMs, elapsedMs)
			if err != nil {
				return 0, err
			}
			return addMilliseconds(posixTime, deltaMs)
		}
		eraDurationMs, err := slotDurationMs(endSlot-beginSlot, slotLengthMs)
		if err != nil {
			return 0, err
		}
		elapsedMs, err = addMilliseconds(elapsedMs, eraDurationMs)
		if err != nil {
			return 0, err
		}
	}

	return 0, fmt.Errorf("slot %d is outside era history", slot)
}

func addPlutusScriptByHash(
	plutusScript lcommon.Script,
	v1ScriptByHash map[string]lcommon.PlutusV1Script,
	v2ScriptByHash map[string]lcommon.PlutusV2Script,
	v3ScriptByHash map[string]lcommon.PlutusV3Script,
	v4ScriptByHash map[string]lcommon.PlutusV4Script,
) {
	switch s := plutusScript.(type) {
	case lcommon.PlutusV1Script:
		hash := s.Hash()
		v1ScriptByHash[hex.EncodeToString(hash[:])] = s
	case *lcommon.PlutusV1Script:
		if s != nil {
			hash := s.Hash()
			v1ScriptByHash[hex.EncodeToString(hash[:])] = *s
		}
	case lcommon.PlutusV2Script:
		hash := s.Hash()
		v2ScriptByHash[hex.EncodeToString(hash[:])] = s
	case *lcommon.PlutusV2Script:
		if s != nil {
			hash := s.Hash()
			v2ScriptByHash[hex.EncodeToString(hash[:])] = *s
		}
	case lcommon.PlutusV3Script:
		hash := s.Hash()
		v3ScriptByHash[hex.EncodeToString(hash[:])] = s
	case *lcommon.PlutusV3Script:
		if s != nil {
			hash := s.Hash()
			v3ScriptByHash[hex.EncodeToString(hash[:])] = *s
		}
	case lcommon.PlutusV4Script:
		hash := s.Hash()
		v4ScriptByHash[hex.EncodeToString(hash[:])] = s
	case *lcommon.PlutusV4Script:
		if s != nil {
			hash := s.Hash()
			v4ScriptByHash[hex.EncodeToString(hash[:])] = *s
		}
	}
}

func buildWitnessDatumMap(
	witnesses lcommon.TransactionWitnessSet,
) map[lcommon.DatumHash]data.PlutusData {
	datumsByHash := make(map[lcommon.DatumHash]data.PlutusData)
	for _, datum := range witnesses.PlutusData() {
		if datum.Data == nil {
			continue
		}
		datumsByHash[datum.Hash()] = datum.Data
	}
	return datumsByHash
}

func resolveDatumData(
	output lcommon.TransactionOutput,
	witnessDatums map[lcommon.DatumHash]data.PlutusData,
) (data.PlutusData, error) {
	datum := output.Datum()
	if datum != nil && datum.Data != nil {
		return datum.Data, nil
	}

	datumHash := output.DatumHash()
	if datumHash == nil {
		if datum == nil {
			return nil, errors.New("datum is missing")
		}
		return nil, errors.New("datum data is nil and datum hash is missing")
	}

	datumData, ok := witnessDatums[*datumHash]
	if !ok || datumData == nil {
		return nil, fmt.Errorf(
			"datum %s not found in witness data",
			hex.EncodeToString(datumHash[:]),
		)
	}
	return datumData, nil
}

type eraHistorySlotState struct {
	systemStartMs int64
	eraHistory    []localstatequery.EraHistoryResult
}

func (s eraHistorySlotState) SlotToTime(slot uint64) (time.Time, error) {
	posixTime, err := slotToPOSIXTime(slot, s.systemStartMs, s.eraHistory)
	if err != nil {
		return time.Time{}, err
	}
	return time.UnixMilli(posixTime).UTC(), nil
}

func (eraHistorySlotState) TimeToSlot(time.Time) (uint64, error) {
	return 0, errors.New("time-to-slot conversion is not supported")
}

type plutusScriptVersion uint

const (
	plutusScriptV1 plutusScriptVersion = iota
	plutusScriptV2
	plutusScriptV3
	plutusScriptV4
)

func resolvedUtxosForScriptContext(
	tx ledger.Transaction,
	resolvedUtxos map[string]ledger.Utxo,
) ([]lcommon.Utxo, error) {
	allInputs := make(
		[]lcommon.TransactionInput,
		0,
		len(tx.Inputs())+len(tx.ReferenceInputs()),
	)
	allInputs = append(allInputs, tx.Inputs()...)
	allInputs = append(allInputs, tx.ReferenceInputs()...)

	seen := make(map[string]struct{}, len(allInputs))
	ret := make([]lcommon.Utxo, 0, len(allInputs))
	for _, input := range allInputs {
		key := input.String()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		utxo, ok := resolvedUtxos[key]
		if !ok {
			return nil, fmt.Errorf("unresolved input %s", key)
		}
		if utxo.Output == nil {
			return nil, fmt.Errorf("resolved input %s has nil output", key)
		}
		ret = append(ret, utxo)
	}
	return ret, nil
}

func buildTxInfoForVersion(
	tx ledger.Transaction,
	resolvedUtxos map[string]ledger.Utxo,
	systemStartMs int64,
	eraHistory []localstatequery.EraHistoryResult,
	protocolMajor uint,
	version plutusScriptVersion,
) (script.TxInfo, error) {
	resolvedInputs, err := resolvedUtxosForScriptContext(tx, resolvedUtxos)
	if err != nil {
		return nil, err
	}
	slotState := eraHistorySlotState{
		systemStartMs: systemStartMs,
		eraHistory:    eraHistory,
	}
	switch version {
	case plutusScriptV1:
		if err := validateLegacyScriptCertificates(tx.Certificates()); err != nil {
			return nil, err
		}
		txInfo, err := script.NewTxInfoV1FromTransaction(
			slotState,
			tx,
			resolvedInputs,
		)
		if err != nil {
			return nil, err
		}
		txInfo.ProtocolMajor = protocolMajor
		return txInfo, nil
	case plutusScriptV2:
		if err := validateLegacyScriptCertificates(tx.Certificates()); err != nil {
			return nil, err
		}
		txInfo, err := script.NewTxInfoV2FromTransaction(
			slotState,
			tx,
			resolvedInputs,
		)
		if err != nil {
			return nil, err
		}
		txInfo.ProtocolMajor = protocolMajor
		return txInfo, nil
	case plutusScriptV3, plutusScriptV4:
		return script.NewTxInfoV3FromTransaction(
			slotState,
			tx,
			resolvedInputs,
		)
	default:
		return nil, fmt.Errorf("unsupported Plutus script version %d", version)
	}
}

func buildScriptContextForVersion(
	tx ledger.Transaction,
	resolvedUtxos map[string]ledger.Utxo,
	purpose script.ScriptPurpose,
	redeemer script.Redeemer,
	systemStartMs int64,
	eraHistory []localstatequery.EraHistoryResult,
	protocolMajor uint,
	version plutusScriptVersion,
) (data.PlutusData, error) {
	txInfo, err := buildTxInfoForVersion(
		tx,
		resolvedUtxos,
		systemStartMs,
		eraHistory,
		protocolMajor,
		version,
	)
	if err != nil {
		return nil, fmt.Errorf("build tx info: %w", err)
	}
	switch version {
	case plutusScriptV1, plutusScriptV2:
		return script.NewScriptContextV1V2(txInfo, purpose).ToPlutusData(), nil
	case plutusScriptV3, plutusScriptV4:
		return script.NewScriptContextV3(
			txInfo,
			redeemer,
			purpose,
		).ToPlutusData(), nil
	default:
		return nil, fmt.Errorf("unsupported Plutus script version %d", version)
	}
}

func buildScriptPurposeForContext(
	tx ledger.Transaction,
	resolvedUtxos map[string]ledger.Utxo,
	redeemerTag lcommon.RedeemerTag,
	redeemerIndex uint32,
	datum data.PlutusData,
) (script.ScriptPurpose, error) {
	switch uint8(redeemerTag) {
	case 0: // Spend
		inputs := sortInputsCanonically(tx.Inputs())
		if int(redeemerIndex) >= len(inputs) {
			return nil, fmt.Errorf(
				"spend redeemer index %d out of range (input count %d)",
				redeemerIndex,
				len(inputs),
			)
		}
		input := inputs[redeemerIndex]
		utxo, ok := resolvedUtxos[input.String()]
		if !ok {
			return nil, fmt.Errorf("UTxO not found for input %s", input.String())
		}
		if utxo.Output == nil {
			return nil, fmt.Errorf("UTxO %s has nil output", input.String())
		}
		return script.ScriptPurposeSpending{
			Input: lcommon.Utxo{
				Id:     input,
				Output: utxo.Output,
			},
			Datum: datum,
		}, nil
	case 1: // Mint
		mint := tx.AssetMint()
		if mint == nil {
			return nil, fmt.Errorf(
				"mint redeemer at index %d but transaction has no minted assets",
				redeemerIndex,
			)
		}
		policies := mint.Policies()
		sortedPolicies := make([]lcommon.Blake2b224, len(policies))
		copy(sortedPolicies, policies)
		sort.Slice(sortedPolicies, func(i, j int) bool {
			return hex.EncodeToString(
				sortedPolicies[i][:],
			) < hex.EncodeToString(
				sortedPolicies[j][:],
			)
		})
		if int(redeemerIndex) >= len(sortedPolicies) {
			return nil, fmt.Errorf(
				"mint redeemer index %d out of range (policy count %d)",
				redeemerIndex,
				len(sortedPolicies),
			)
		}
		return script.ScriptPurposeMinting{
			PolicyId: sortedPolicies[redeemerIndex],
		}, nil
	default:
		return nil, fmt.Errorf(
			"script purpose not supported yet for redeemer tag %d (index %d)",
			redeemerTag,
			redeemerIndex,
		)
	}
}

func maxTxExUnitsFromProtocolParams(
	protoParams lcommon.ProtocolParameters,
) (lcommon.ExUnits, error) {
	switch params := protoParams.(type) {
	case *ledger.DijkstraProtocolParameters:
		return params.MaxTxExUnits, nil
	case *ledger.ConwayProtocolParameters:
		return params.MaxTxExUnits, nil
	case *ledger.BabbageProtocolParameters:
		return params.MaxTxExUnits, nil
	case *ledger.AlonzoProtocolParameters:
		return params.MaxTxExUnits, nil
	default:
		return lcommon.ExUnits{}, fmt.Errorf(
			"unsupported protocol parameters type for max tx ex units: %T",
			protoParams,
		)
	}
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

func validateLegacyScriptCertificates(certs []lcommon.Certificate) error {
	for _, cert := range certs {
		switch cert.(type) {
		case *lcommon.StakeRegistrationCertificate,
			*lcommon.RegistrationCertificate,
			*lcommon.StakeDeregistrationCertificate,
			*lcommon.DeregistrationCertificate,
			*lcommon.StakeDelegationCertificate,
			*lcommon.PoolRegistrationCertificate,
			*lcommon.PoolRetirementCertificate:
		default:
			return fmt.Errorf(
				"unsupported certificate type %T for script evaluation",
				cert,
			)
		}
	}
	return nil
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
		// For larger integers, use BigUInt for positive and CBOR-style
		// BigNInt for negative.
		if v.Inner.Sign() < 0 {
			negativeBignum := new(big.Int).Neg(v.Inner)
			negativeBignum.Sub(negativeBignum, big.NewInt(1))
			return &cardano.PlutusData{
				PlutusData: &cardano.PlutusData_BigInt{
					BigInt: &cardano.BigInt{
						BigInt: &cardano.BigInt_BigNInt{
							BigNInt: negativeBignum.Bytes(),
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
	maxBudget, err := maxTxExUnitsFromProtocolParams(protoParams)
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
	eraHistory, err := oConn.LocalStateQuery().Client.GetEraHistory()
	if err != nil {
		return connect.NewResponse(resp), fmt.Errorf(
			"get era history: %w",
			err,
		)
	}

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
		// A transaction with no redeemers has no Plutus scripts to
		// evaluate (e.g. a simple payment, or any pre-Alonzo transaction).
		// This is not an error: report an empty evaluation result, matching
		// the output produced when redeemers are present but empty.
		resp.Report = &submit.AnyChainEval{
			Chain: &submit.AnyChainEval_Cardano{
				Cardano: &cardano.TxEval{
					Fee: &cardano.BigInt{
						BigInt: &cardano.BigInt_Int{
							Int: 0,
						},
					},
					Redeemers: []*cardano.Redeemer{},
				},
			},
		}
		return connect.NewResponse(resp), nil
	}
	// Only spend and mint redeemers can be evaluated so far. Fail fast
	// rather than returning misleading zero-cost results for the rest.
	for key := range redeemers.Iter() {
		if key.Tag != lcommon.RedeemerTagSpend &&
			key.Tag != lcommon.RedeemerTagMint {
			return connect.NewResponse(resp), fmt.Errorf(
				"evaluation of redeemer tag %d is not supported yet (index %d): only spend and mint redeemers are supported",
				key.Tag,
				key.Index,
			)
		}
	}
	witnessDatums := buildWitnessDatumMap(witnesses)

	// Get cost models and protocol version from protocol parameters
	var costModels map[uint][]int64
	var protoVersionMajor, protoVersionMinor uint
	switch params := protoParams.(type) {
	case *ledger.DijkstraProtocolParameters:
		costModels = params.CostModels
		protoVersionMajor = params.ProtocolVersion.Major
		protoVersionMinor = params.ProtocolVersion.Minor
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
	v4ScriptByHash := make(map[string]lcommon.PlutusV4Script)

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
	for _, s := range lcommon.PlutusV4ScriptsFromWitnessSet(witnesses) {
		hash := s.Hash()
		v4ScriptByHash[hex.EncodeToString(hash[:])] = s
	}
	for _, utxo := range resolvedUtxos {
		if utxo.Output == nil {
			continue
		}
		addPlutusScriptByHash(
			utxo.Output.ScriptRef(),
			v1ScriptByHash,
			v2ScriptByHash,
			v3ScriptByHash,
			v4ScriptByHash,
		)
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
		var datumData data.PlutusData

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
				// A spend redeemer must reference a script-locked input.
				// If the resolved input is key-locked there is no script to
				// evaluate; emitting zero execution units would under-report
				// the budget, so fail instead.
				return connect.NewResponse(resp), fmt.Errorf(
					"spend redeemer tag=%d index=%d references input %s with a key address, not a script address",
					key.Tag,
					key.Index,
					input.String(),
				)
			}

			datumData, err = resolveDatumData(utxo.Output, witnessDatums)
			if err != nil {
				return connect.NewResponse(
						resp,
					), fmt.Errorf(
						"resolve datum for input %s: %w",
						input.String(),
						err,
					)
			}
		} else if uint8(key.Tag) == 1 { // RedeemerTagMint
			// For minting, the script hash is the policy ID
			mint := tx.AssetMint()
			if mint == nil {
				// A mint redeemer must correspond to an asset being minted.
				// Without a mint there is no policy script to evaluate;
				// emitting zero execution units would under-report the
				// budget, so fail instead.
				return connect.NewResponse(resp), fmt.Errorf(
					"mint redeemer tag=%d index=%d has no corresponding mint in the transaction",
					key.Tag,
					key.Index,
				)
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
		} else {
			// Unreachable: non-spend/mint redeemers are rejected before
			// this loop
			return connect.NewResponse(resp), fmt.Errorf(
				"evaluation of redeemer tag %d is not supported yet (index %d)",
				key.Tag,
				key.Index,
			)
		}
		redeemerData := value.Data.Data
		scriptPurpose, err := buildScriptPurposeForContext(
			tx,
			resolvedUtxos,
			key.Tag,
			key.Index,
			datumData,
		)
		if err != nil {
			return connect.NewResponse(resp), fmt.Errorf(
				"build script purpose for redeemer tag=%d index=%d: %w",
				key.Tag,
				key.Index,
				err,
			)
		}
		contextRedeemer := script.Redeemer{
			Tag:     key.Tag,
			Index:   key.Index,
			Data:    redeemerData,
			ExUnits: value.ExUnits,
		}

		// Look up script by hash and evaluate using gouroboros methods
		var exUnits lcommon.ExUnits
		var evalErr error

		if v1Script, found := v1ScriptByHash[scriptHashHex]; found {
			// V1 script: Evaluate(datum, redeemer, scriptContext, budget, evalContext)
			costModelData, ok := costModels[0]
			if !ok {
				return connect.NewResponse(resp), fmt.Errorf(
					"missing Plutus V1 cost model for redeemer tag=%d index=%d script_hash=%s",
					key.Tag,
					key.Index,
					scriptHashHex,
				)
			}
			langVersion := cek.LanguageVersionV1
			evalContext, err := cek.NewEvalContext(
				langVersion,
				protoVersion,
				costModelData,
			)
			if err != nil {
				return connect.NewResponse(resp), fmt.Errorf(
					"create Plutus V1 eval context for redeemer tag=%d index=%d script_hash=%s: %w",
					key.Tag,
					key.Index,
					scriptHashHex,
					err,
				)
			}
			contextData, err := buildScriptContextForVersion(
				tx,
				resolvedUtxos,
				scriptPurpose,
				contextRedeemer,
				systemStartMs,
				eraHistory,
				protoVersionMajor,
				plutusScriptV1,
			)
			if err != nil {
				return connect.NewResponse(resp), fmt.Errorf(
					"build Plutus V1 script context for redeemer tag=%d index=%d script_hash=%s: %w",
					key.Tag,
					key.Index,
					scriptHashHex,
					err,
				)
			}
			exUnits, evalErr = v1Script.Evaluate(
				datumData,
				redeemerData,
				contextData,
				maxBudget,
				evalContext,
			)
		} else if v2Script, found := v2ScriptByHash[scriptHashHex]; found {
			// V2 script: Evaluate(datum, redeemer, scriptContext, budget, evalContext)
			costModelData, ok := costModels[1]
			if !ok {
				return connect.NewResponse(resp), fmt.Errorf(
					"missing Plutus V2 cost model for redeemer tag=%d index=%d script_hash=%s",
					key.Tag,
					key.Index,
					scriptHashHex,
				)
			}
			langVersion := cek.LanguageVersionV2
			evalContext, err := cek.NewEvalContext(
				langVersion,
				protoVersion,
				costModelData,
			)
			if err != nil {
				return connect.NewResponse(resp), fmt.Errorf(
					"create Plutus V2 eval context for redeemer tag=%d index=%d script_hash=%s: %w",
					key.Tag,
					key.Index,
					scriptHashHex,
					err,
				)
			}
			contextData, err := buildScriptContextForVersion(
				tx,
				resolvedUtxos,
				scriptPurpose,
				contextRedeemer,
				systemStartMs,
				eraHistory,
				protoVersionMajor,
				plutusScriptV2,
			)
			if err != nil {
				return connect.NewResponse(resp), fmt.Errorf(
					"build Plutus V2 script context for redeemer tag=%d index=%d script_hash=%s: %w",
					key.Tag,
					key.Index,
					scriptHashHex,
					err,
				)
			}
			exUnits, evalErr = v2Script.Evaluate(
				datumData,
				redeemerData,
				contextData,
				maxBudget,
				evalContext,
			)
		} else if v3Script, found := v3ScriptByHash[scriptHashHex]; found {
			// V3 script: Evaluate(scriptContext, budget, evalContext)
			costModelData, ok := costModels[2]
			if !ok {
				return connect.NewResponse(resp), fmt.Errorf(
					"missing Plutus V3 cost model for redeemer tag=%d index=%d script_hash=%s",
					key.Tag,
					key.Index,
					scriptHashHex,
				)
			}
			langVersion := cek.LanguageVersionV3
			evalContext, err := cek.NewEvalContext(
				langVersion,
				protoVersion,
				costModelData,
			)
			if err != nil {
				return connect.NewResponse(resp), fmt.Errorf(
					"create Plutus V3 eval context for redeemer tag=%d index=%d script_hash=%s: %w",
					key.Tag,
					key.Index,
					scriptHashHex,
					err,
				)
			}
			contextData, err := buildScriptContextForVersion(
				tx,
				resolvedUtxos,
				scriptPurpose,
				contextRedeemer,
				systemStartMs,
				eraHistory,
				protoVersionMajor,
				plutusScriptV3,
			)
			if err != nil {
				return connect.NewResponse(resp), fmt.Errorf(
					"build Plutus V3 script context for redeemer tag=%d index=%d script_hash=%s: %w",
					key.Tag,
					key.Index,
					scriptHashHex,
					err,
				)
			}
			exUnits, evalErr = v3Script.Evaluate(
				contextData,
				maxBudget,
				evalContext,
			)
		} else if v4Script, found := v4ScriptByHash[scriptHashHex]; found {
			// V4 script: Evaluate(scriptContext, budget, evalContext)
			costModelData, ok := costModels[3]
			if !ok {
				return connect.NewResponse(resp), fmt.Errorf(
					"missing Plutus V4 cost model for redeemer tag=%d index=%d script_hash=%s",
					key.Tag,
					key.Index,
					scriptHashHex,
				)
			}
			langVersion := cek.LanguageVersionV4
			evalContext, err := cek.NewEvalContext(
				langVersion,
				protoVersion,
				costModelData,
			)
			if err != nil {
				return connect.NewResponse(resp), fmt.Errorf(
					"create Plutus V4 eval context for redeemer tag=%d index=%d script_hash=%s: %w",
					key.Tag,
					key.Index,
					scriptHashHex,
					err,
				)
			}
			contextData, err := buildScriptContextForVersion(
				tx,
				resolvedUtxos,
				scriptPurpose,
				contextRedeemer,
				systemStartMs,
				eraHistory,
				protoVersionMajor,
				plutusScriptV4,
			)
			if err != nil {
				return connect.NewResponse(resp), fmt.Errorf(
					"build Plutus V4 script context for redeemer tag=%d index=%d script_hash=%s: %w",
					key.Tag,
					key.Index,
					scriptHashHex,
					err,
				)
			}
			exUnits, evalErr = v4Script.Evaluate(
				contextData,
				maxBudget,
				evalContext,
			)
		} else {
			// A spend or mint redeemer requires its Plutus script to be
			// present in the witness set or referenced via a reference
			// input. If it cannot be found we cannot evaluate it; returning
			// zero execution units here would under-report the budget and
			// mislead downstream fee/build logic, so fail instead.
			return connect.NewResponse(resp), fmt.Errorf(
				"script not found for redeemer tag=%d index=%d script_hash=%s",
				key.Tag,
				key.Index,
				scriptHashHex,
			)
		}

		if evalErr != nil {
			return connect.NewResponse(resp), fmt.Errorf(
				"evaluate script for redeemer tag=%d index=%d: %w",
				key.Tag,
				key.Index,
				evalErr,
			)
		}
		redeemer.ExUnits.Steps = uint64(exUnits.Steps)   //nolint:gosec
		redeemer.ExUnits.Memory = uint64(exUnits.Memory) //nolint:gosec

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
