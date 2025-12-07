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

	connect "connectrpc.com/connect"
	"github.com/blinklabs-io/adder/event"
	"github.com/blinklabs-io/cardano-node-api/internal/node"
	"github.com/blinklabs-io/gouroboros/ledger"
	ocommon "github.com/blinklabs-io/gouroboros/protocol/common"
	"github.com/blinklabs-io/plutigo/cek"
	"github.com/blinklabs-io/plutigo/data"
	"github.com/blinklabs-io/plutigo/syn"
	"github.com/utxorpc/go-codegen/utxorpc/v1alpha/cardano"
	submit "github.com/utxorpc/go-codegen/utxorpc/v1alpha/submit"
	"github.com/utxorpc/go-codegen/utxorpc/v1alpha/submit/submitconnect"
)

// submitServiceServer implements the SubmitService API
type submitServiceServer struct {
	submitconnect.UnimplementedSubmitServiceHandler
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

// evaluateScript evaluates a Plutus script with the given arguments
func evaluateScript(
	scriptBytes []byte,
	args []data.PlutusData,
) (uint64, uint64, error) {
	// Decode the script
	program, err := syn.Decode[syn.DeBruijn](scriptBytes)
	if err != nil {
		return 0, 0, fmt.Errorf("decode script: %w", err)
	}

	// Apply arguments to the script
	term := program.Term
	for _, arg := range args {
		term = &syn.Apply[syn.DeBruijn]{
			Function: term,
			Argument: &syn.Constant{
				Con: &syn.Data{Inner: arg},
			},
		}
	}

	// Create machine with version-based cost model
	machine := cek.NewMachineWithVersionCosts[syn.DeBruijn](
		program.Version,
		200,
	)

	_, err = machine.Run(term)
	if err != nil {
		return 0, 0, fmt.Errorf("execute script: %w", err)
	}

	// Safe conversion: ExBudget should not be negative, but check to avoid overflow
	var steps, memory uint64
	if machine.ExBudget.Cpu < 0 {
		steps = 0
	} else {
		steps = uint64(machine.ExBudget.Cpu) //nolint:gosec
	}
	if machine.ExBudget.Mem < 0 {
		memory = 0
	} else {
		memory = uint64(machine.ExBudget.Mem) //nolint:gosec
	}
	return steps, memory, nil //nolint:gosec
}

// convertPlutusData converts plutigo data.PlutusData to *cardano.PlutusData
func convertPlutusData(pd data.PlutusData) *cardano.PlutusData {
	switch v := pd.(type) {
	case *data.Constr:
		fields := make([]*cardano.PlutusData, len(v.Fields))
		for i, field := range v.Fields {
			fields[i] = convertPlutusData(field)
		}
		return &cardano.PlutusData{
			PlutusData: &cardano.PlutusData_Constr{
				Constr: &cardano.Constr{
					Tag:    uint32(v.Tag), //nolint:gosec
					Fields: fields,
				},
			},
		}
	case *data.Map:
		pairs := make([]*cardano.PlutusDataPair, len(v.Pairs))
		for i, pair := range v.Pairs {
			pairs[i] = &cardano.PlutusDataPair{
				Key:   convertPlutusData(pair[0]),
				Value: convertPlutusData(pair[1]),
			}
		}
		return &cardano.PlutusData{
			PlutusData: &cardano.PlutusData_Map{
				Map: &cardano.PlutusDataMap{
					Pairs: pairs,
				},
			},
		}
	case *data.Integer:
		return &cardano.PlutusData{
			PlutusData: &cardano.PlutusData_BigInt{
				BigInt: &cardano.BigInt{
					BigInt: &cardano.BigInt_Int{
						Int: v.Inner.Int64(),
					},
				},
			},
		}
	case *data.ByteString:
		return &cardano.PlutusData{
			PlutusData: &cardano.PlutusData_BoundedBytes{
				BoundedBytes: v.Inner,
			},
		}
	case *data.List:
		items := make([]*cardano.PlutusData, len(v.Items))
		for i, item := range v.Items {
			items[i] = convertPlutusData(item)
		}
		return &cardano.PlutusData{
			PlutusData: &cardano.PlutusData_Array{
				Array: &cardano.PlutusDataArray{
					Items: items,
				},
			},
		}
	default:
		// Should not happen
		return nil
	}
}

// EvalTx
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

	// Get cost models from protocol parameters
	var costModels map[uint][]int64
	if conwayParams, ok := protoParams.(*ledger.ConwayProtocolParameters); ok {
		costModels = conwayParams.CostModels
	}

	// Get scripts
	v1Scripts := witnesses.PlutusV1Scripts()
	v2Scripts := witnesses.PlutusV2Scripts()
	v3Scripts := witnesses.PlutusV3Scripts()
	allScripts := make(
		[][]byte,
		0,
		len(v1Scripts)+len(v2Scripts)+len(v3Scripts),
	)
	scriptVersions := make(
		[]uint,
		0,
		len(v1Scripts)+len(v2Scripts)+len(v3Scripts),
	)
	for _, s := range v1Scripts {
		allScripts = append(allScripts, s.RawScriptBytes())
		scriptVersions = append(scriptVersions, 1)
	}
	for _, s := range v2Scripts {
		allScripts = append(allScripts, s.RawScriptBytes())
		scriptVersions = append(scriptVersions, 2)
	}
	for _, s := range v3Scripts {
		allScripts = append(allScripts, s.RawScriptBytes())
		scriptVersions = append(scriptVersions, 3)
	}

	// Count redeemers for pre-allocation
	redeemerCount := 0
	for range redeemers.Iter() {
		redeemerCount++
	}

	txEvalRedeemers := make([]*cardano.Redeemer, 0, redeemerCount)
	for key, value := range redeemers.Iter() {
		redeemer := &cardano.Redeemer{
			Purpose: cardano.RedeemerPurpose(key.Tag),
			Payload: convertPlutusData(value.Data.Data),
			Index:   key.Index,
			ExUnits: &cardano.ExUnits{
				Steps:  0,
				Memory: 0,
			},
		}

		// Try to evaluate if we have a script for this redeemer
		if int(key.Index) < len(allScripts) {
			scriptBytes := allScripts[key.Index]
			version := scriptVersions[key.Index]

			// Build arguments based on purpose
			var args []data.PlutusData

			// For now, only handle spending scripts with datum, redeemer, context
			if key.Tag == 0 { // RedeemerTagSpend
				// Get datum - for simplicity, assume it's the redeemer data for now
				datum := value.Data.Data
				redeemerData := value.Data.Data
				// TODO: build proper script context
				contextData := data.NewConstr(0) // dummy context

				args = []data.PlutusData{datum, redeemerData, contextData}
			} else {
				// For other purposes, just redeemer and context
				redeemerData := value.Data.Data
				contextData := data.NewConstr(0) // dummy context
				args = []data.PlutusData{redeemerData, contextData}
			}

			// Filter cost model for this version
			versionCostModel := make(map[uint][]int64)
			if model, ok := costModels[version]; ok {
				versionCostModel[version] = model
			}

			steps, memory, err := evaluateScript(
				scriptBytes,
				args,
			)
			if err != nil {
				log.Printf(
					"Failed to evaluate script for redeemer %d: %v",
					key.Index,
					err,
				)
			} else {
				redeemer.ExUnits.Steps = steps
				redeemer.ExUnits.Memory = memory
			}
		}

		txEvalRedeemers = append(txEvalRedeemers, redeemer)
	}

	resp.Report = &submit.AnyChainEval{
		Chain: &submit.AnyChainEval_Cardano{
			Cardano: &cardano.TxEval{
				Fee:       nil, // TODO: set fee
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
