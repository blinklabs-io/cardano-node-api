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
	input_chainsync "github.com/blinklabs-io/adder/input/chainsync"
	"github.com/blinklabs-io/gouroboros/ledger"
	ocommon "github.com/blinklabs-io/gouroboros/protocol/common"
	submit "github.com/utxorpc/go-codegen/utxorpc/v1alpha/submit"
	"github.com/utxorpc/go-codegen/utxorpc/v1alpha/submit/submitconnect"

	"github.com/blinklabs-io/cardano-node-api/internal/node"
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

	// txRawList
	txRawList := req.Msg.GetTx() // []*AnyChainTx
	log.Printf("Got a SubmitTx request with %d transactions", len(txRawList))
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

	// Loop through the transactions and submit each
	errorList := make([]error, len(txRawList))
	hasError := false
	for i, txi := range txRawList {
		txRawBytes := txi.GetRaw() // raw bytes
		txType, err := ledger.DetermineTransactionType(txRawBytes)
		placeholderRef := []byte{}
		if err != nil {
			resp.Ref = append(resp.Ref, placeholderRef)
			errorList[i] = err
			hasError = true
			continue
		}
		tx, err := ledger.NewTransactionFromCbor(txType, txRawBytes)
		if err != nil {
			resp.Ref = append(resp.Ref, placeholderRef)
			errorList[i] = err
			hasError = true
			continue
		}
		// Submit the transaction
		err = oConn.LocalTxSubmission().Client.SubmitTx(
			uint16(txType), // #nosec G115
			txRawBytes,
		)
		if err != nil {
			resp.Ref = append(resp.Ref, placeholderRef)
			errorList[i] = fmt.Errorf("%s", err.Error())
			hasError = true
			continue
		}
		txHexBytes, err := hex.DecodeString(tx.Hash())
		if err != nil {
			resp.Ref = append(resp.Ref, placeholderRef)
			errorList[i] = err
			hasError = true
			continue
		}
		resp.Ref = append(resp.Ref, txHexBytes)
	}
	if hasError {
		return connect.NewResponse(resp), fmt.Errorf("%v", errorList)
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
			case input_chainsync.TransactionEvent:
				logger.Debug("Received TransactionEvent", "hash", v.Transaction.Hash())
				for _, r := range ref {
					refHash := hex.EncodeToString(r)
					eventHash := v.Transaction.Hash()

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
		cTx := tx.Utxorpc() // *cardano.Tx
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
