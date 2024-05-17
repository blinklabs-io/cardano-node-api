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
	// "fmt"
	"log"

	connect "connectrpc.com/connect"
	"github.com/blinklabs-io/gouroboros/ledger"
	// ocommon "github.com/blinklabs-io/gouroboros/protocol/common"
	query "github.com/utxorpc/go-codegen/utxorpc/v1alpha/query"
	"github.com/utxorpc/go-codegen/utxorpc/v1alpha/query/queryconnect"

	"github.com/blinklabs-io/cardano-node-api/internal/node"
)

// queryServiceServer implements the WatchService API
type queryServiceServer struct {
	queryconnect.UnimplementedQueryServiceHandler
}

// ReadParams
func (s *queryServiceServer) ReadParams(
	ctx context.Context,
	req *connect.Request[query.ReadParamsRequest],
) (*connect.Response[query.ReadParamsResponse], error) {

	fieldMask := req.Msg.GetFieldMask()
	log.Printf("Got a ReadParams request with fieldMask %v", fieldMask)
	resp := &query.ReadParamsResponse{}

	// Connect to node
	oConn, err := node.GetConnection(nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		// Close Ouroboros connection
		oConn.Close()
	}()
	// Start client
	oConn.LocalStateQuery().Client.Start()

	// Get protoParams
	protoParams, err := oConn.LocalStateQuery().Client.GetCurrentProtocolParams()
	if err != nil {
		log.Printf("ERROR: %s", err)
		return nil, err
	}

	// Get chain point (slot and hash)
	point, err := oConn.LocalStateQuery().Client.GetChainPoint()
	if err != nil {
		log.Printf("ERROR: %s", err)
		return nil, err
	}
	var acp query.AnyChainParams
	// var acpc query.AnyChainParams_Cardano
	acpc := &query.AnyChainParams_Cardano{}
	log.Printf("protocol parameters: %v", protoParams)
	// acpc.Cardano = protoParams.Utxorpc()
	resp.LedgerTip = &query.ChainPoint{
		Slot: point.Slot,
		Hash: point.Hash,
	}
	// acp.Params = &acpc
	acp.Params = acpc
	resp.Values = &acp
	return connect.NewResponse(resp), nil
}

// ReadUtxos
func (s *queryServiceServer) ReadUtxos(
	ctx context.Context,
	req *connect.Request[query.ReadUtxosRequest],
) (*connect.Response[query.ReadUtxosResponse], error) {

	keys := req.Msg.GetKeys() // []*TxoRef
	log.Printf("Got a ReadUtxos request with keys %v", keys)
	resp := &query.ReadUtxosResponse{}

	// Connect to node
	oConn, err := node.GetConnection(nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		// Close Ouroboros connection
		oConn.Close()
	}()
	// Start client
	oConn.LocalStateQuery().Client.Start()

	// Setup our query input
	var tmpTxIns []ledger.TransactionInput
	for _, txo := range keys {
		// txo.Hash, txo.Index
		tmpTxIn := ledger.ShelleyTransactionInput{
			TxId: ledger.Blake2b256(txo.Hash),
			OutputIndex: uint32(txo.Index),
		}
		tmpTxIns = append(tmpTxIns, tmpTxIn)
	}

	// Get UTxOs
	utxos, err := oConn.LocalStateQuery().Client.GetUTxOByTxIn(tmpTxIns)
	if err != nil {
		log.Printf("ERROR: %s", err)
		return nil, err
	}

	// Get chain point (slot and hash)
	point, err := oConn.LocalStateQuery().Client.GetChainPoint()
	if err != nil {
		log.Printf("ERROR: %s", err)
		return nil, err
	}

	for _, txo := range keys {
		for utxoId, utxo := range utxos.Results {
			var aud query.AnyUtxoData
			var audc query.AnyUtxoData_Cardano
			aud.TxoRef = txo
			txHash := hex.EncodeToString(txo.Hash)
			if utxoId.Hash.String() == txHash && uint32(utxoId.Idx) == txo.Index {
				aud.NativeBytes = utxo.Cbor()
				audc.Cardano = utxo.Utxorpc()
				aud.ParsedState = &audc
			}
			resp.Items = append(resp.Items, &aud)
		}
	}
	resp.LedgerTip = &query.ChainPoint{
		Slot: point.Slot,
		Hash: point.Hash,
	}
	return connect.NewResponse(resp), nil
}

// SearchUtxos
func (s *queryServiceServer) SearchUtxos(
	ctx context.Context,
	req *connect.Request[query.SearchUtxosRequest],
) (*connect.Response[query.SearchUtxosResponse], error) {
	return nil, nil
}

// StreamUtxos
