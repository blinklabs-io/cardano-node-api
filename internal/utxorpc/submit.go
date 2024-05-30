// Copyright 2024 Blink Labs Software
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
	"fmt"
	"log"

	connect "connectrpc.com/connect"
	"github.com/blinklabs-io/adder/event"
	input_chainsync "github.com/blinklabs-io/adder/input/chainsync"
	"github.com/blinklabs-io/gouroboros/ledger"
	ocommon "github.com/blinklabs-io/gouroboros/protocol/common"
	submit "github.com/utxorpc/go-codegen/utxorpc/v1alpha/submit"
	"github.com/utxorpc/go-codegen/utxorpc/v1alpha/submit/submitconnect"
	"golang.org/x/crypto/blake2b"

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
			uint16(txType),
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

// WaitForTx
func (s *submitServiceServer) WaitForTx(
	ctx context.Context,
	req *connect.Request[submit.WaitForTxRequest],
	stream *connect.ServerStream[submit.WaitForTxResponse],
) error {

	ref := req.Msg.GetRef() // [][]byte
	log.Printf("Got a WaitForTx request with %d transactions", len(ref))

	// Setup event channel
	eventChan := make(chan event.Event, 10)
	connCfg := node.ConnectionConfig{
		ChainSyncEventChan: eventChan,
	}
	// Connect to node
	oConn, err := node.GetConnection(&connCfg)
	if err != nil {
		return err
	}
	defer func() {
		// Close Ouroboros connection
		oConn.Close()
	}()

	// Get our starting point
	var point ocommon.Point
	tip, err := oConn.ChainSync().Client.GetCurrentTip()
	if err != nil {
		log.Printf("ERROR: %s", err)
		return err
	}
	point = tip.Point

	// Start the sync with the node
	err = oConn.ChainSync().Client.Sync([]ocommon.Point{point})
	if err != nil {
		log.Printf("ERROR: %s", err)
		return err
	}

	// Wait for events
	for {
		evt, ok := <-eventChan
		if !ok {
			log.Printf("ERROR: channel closed")
			return fmt.Errorf("ERROR: channel closed")
		}

		switch v := evt.Payload.(type) {
		case input_chainsync.TransactionEvent:
			for _, r := range ref {
				resp := &submit.WaitForTxResponse{}
				resp.Ref = r
				resp.Stage = submit.Stage_STAGE_UNSPECIFIED
				tc := evt.Context.(input_chainsync.TransactionContext)
				// taken from gOuroboros generateTransactionHash
				tmpHash, err := blake2b.New256(nil)
				if err != nil {
					return err
				}
				tmpHash.Write(r)
				txHash := hex.EncodeToString(tmpHash.Sum(nil))
				// Compare against our event's hash
				if txHash == v.Transaction.Hash() {
					resp.Stage = submit.Stage_STAGE_CONFIRMED
					// Send it!
					err = stream.Send(resp)
					if err != nil {
						return err
					}
					log.Printf(
						"transaction: id: %d, hash: %s",
						tc.TransactionIdx,
						tc.TransactionHash,
					)
				}
			}
		}
	}
}

// ReadMempool
func (s *submitServiceServer) ReadMempool(
	ctx context.Context,
	req *connect.Request[submit.ReadMempoolRequest],
) (*connect.Response[submit.ReadMempoolResponse], error) {

	// This is GetTxs until https://github.com/utxorpc/spec/pull/95
	txim := req.Msg.GetTxs() // []*TxInMempool
	log.Printf("Got a ReadMempool request with %d transactions", len(txim))
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
		var act submit.AnyChainTx
		var actr submit.AnyChainTx_Raw
		actr.Raw = txRawBytes
		act.Type = &actr
		record := &submit.TxInMempool{
			Tx:    &act,
			Stage: submit.Stage_STAGE_MEMPOOL,
		}
		mempool = append(mempool, record)
	}

	// Check each requested Tx against our mempool
	for _, txi := range txim {
		txi.Stage = submit.Stage_STAGE_UNSPECIFIED
		for _, tx := range mempool {
			if txi.Stage == submit.Stage_STAGE_MEMPOOL {
				break
			}
			if txi.Tx.String() == tx.Tx.String() {
				txi.Stage = submit.Stage_STAGE_MEMPOOL
			}
		}
		resp.Stage = append(resp.Stage, txi.Stage)
	}
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
		var act submit.AnyChainTx
		var actr submit.AnyChainTx_Raw
		actr.Raw = txRawBytes
		act.Type = &actr
		record := &submit.TxInMempool{
			Tx:    &act,
			Stage: submit.Stage_STAGE_MEMPOOL,
		}
		resp.Tx = record
		if record.Tx.String() == cTx.String() {
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
