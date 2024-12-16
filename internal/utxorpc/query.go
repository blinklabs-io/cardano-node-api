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
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"log"

	connect "connectrpc.com/connect"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/ledger/common"

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
		return nil, err
	}

	// Get chain point (slot and hash)
	point, err := oConn.LocalStateQuery().Client.GetChainPoint()
	if err != nil {
		return nil, err
	}

	// Set up response parameters
	acpc := &query.AnyChainParams_Cardano{
		Cardano: protoParams.Utxorpc(),
	}

	resp.LedgerTip = &query.ChainPoint{
		Slot: point.Slot,
		Hash: point.Hash,
	}
	resp.Values = &query.AnyChainParams{
		Params: acpc,
	}

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
			TxId:        ledger.Blake2b256(txo.Hash),
			OutputIndex: uint32(txo.Index),
		}
		tmpTxIns = append(tmpTxIns, tmpTxIn)
	}

	// Get UTxOs
	utxos, err := oConn.LocalStateQuery().Client.GetUTxOByTxIn(tmpTxIns)
	if err != nil {
		return nil, err
	}

	// Get chain point (slot and hash)
	point, err := oConn.LocalStateQuery().Client.GetChainPoint()
	if err != nil {
		return nil, err
	}

	for _, txo := range keys {
		for utxoId, utxo := range utxos.Results {
			var aud query.AnyUtxoData
			var audc query.AnyUtxoData_Cardano
			aud.TxoRef = txo
			txHash := hex.EncodeToString(txo.Hash)
			if utxoId.Hash.String() == txHash &&
				uint32(utxoId.Idx) == txo.Index {
				aud.NativeBytes = utxo.Cbor()
				audc.Cardano = utxo.Utxorpc()
				if audc.Cardano.Datum != nil {
					// Check if Datum.Hash is all zeroes
					isAllZeroes := true
					for _, b := range audc.Cardano.Datum.Hash {
						if b != 0 {
							isAllZeroes = false
							break
						}
					}
					if isAllZeroes {
						// No actual datum; set Datum to nil to omit it
						audc.Cardano.Datum = nil
						log.Print(
							"Datum Hash is all zeroes; setting Datum to nil",
						)
					} else {
						log.Printf("Datum Hash present: %x", audc.Cardano.Datum.Hash)
					}
				}
				aud.ParsedState = &audc
			}
			resp.Items = append(resp.Items, &aud)
		}
	}
	resp.LedgerTip = &query.ChainPoint{
		Slot: point.Slot,
		Hash: point.Hash,
	}
	log.Printf(
		"Prepared response with LedgerTip: Slot=%v, Hash=%v",
		resp.LedgerTip.Slot,
		resp.LedgerTip.Hash,
	)
	log.Printf("Final response: %v", resp)
	return connect.NewResponse(resp), nil
}

// SearchUtxos
func (s *queryServiceServer) SearchUtxos(
	ctx context.Context,
	req *connect.Request[query.SearchUtxosRequest],
) (*connect.Response[query.SearchUtxosResponse], error) {

	predicate := req.Msg.GetPredicate() // UtxoPredicate
	log.Printf("Got a SearchUtxos request with predicate %v", predicate)
	resp := &query.SearchUtxosResponse{}

	if predicate == nil {
		return nil, fmt.Errorf("ERROR: empty predicate: %v", predicate)
	}

	addressPattern := predicate.GetMatch().GetCardano().GetAddress()
	assetPattern := predicate.GetMatch().GetCardano().GetAsset()

	var addresses []common.Address
	if addressPattern != nil {
		// Handle Exact Address
		exactAddressBytes := addressPattern.GetExactAddress()
		if exactAddressBytes != nil {
			var addr common.Address
			err := addr.UnmarshalCBOR(exactAddressBytes)
			if err != nil {
				return nil, fmt.Errorf(
					"failed to decode exact address: %w",
					err,
				)
			}
			addresses = append(addresses, addr)
		}

		// Handle Payment Part
		paymentPart := addressPattern.GetPaymentPart()
		if paymentPart != nil {
			log.Printf("PaymentPart is present, decoding...")
			var paymentAddr common.Address
			err := paymentAddr.UnmarshalCBOR(paymentPart)
			if err != nil {
				return nil, fmt.Errorf("failed to decode payment part: %w", err)
			}
			addresses = append(addresses, paymentAddr)
		}

		// Handle Delegation Part
		delegationPart := addressPattern.GetDelegationPart()
		if delegationPart != nil {
			log.Printf("DelegationPart is present, decoding...")
			var delegationAddr common.Address
			err := delegationAddr.UnmarshalCBOR(delegationPart)
			if err != nil {
				return nil, fmt.Errorf(
					"failed to decode delegation part: %w",
					err,
				)
			}
			addresses = append(addresses, delegationAddr)
		}
	}

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

	// Get UTxOs
	utxos, err := oConn.LocalStateQuery().Client.GetUTxOByAddress(addresses)
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

	// Proceed to include the UTxO in the response
	for utxoId, utxo := range utxos.Results {
		var aud query.AnyUtxoData
		var audc query.AnyUtxoData_Cardano
		aud.TxoRef = &query.TxoRef{
			Hash:  utxoId.Hash.Bytes(),
			Index: uint32(utxoId.Idx),
		}
		aud.NativeBytes = utxo.Cbor()
		audc.Cardano = utxo.Utxorpc()
		aud.ParsedState = &audc

		// If AssetPattern is specified, filter based on it
		if assetPattern != nil {
			assetFound := false
			for _, multiasset := range audc.Cardano.Assets {
				if bytes.Equal(multiasset.PolicyId, assetPattern.PolicyId) {
					for _, asset := range multiasset.Assets {
						if bytes.Equal(asset.Name, assetPattern.AssetName) {
							assetFound = true
							break
						}
					}
				}
				if assetFound {
					break
				}
			}

			// Asset not found; skip this UTxO
			if !assetFound {
				continue
			}
		}
		resp.Items = append(resp.Items, &aud)
	}

	resp.LedgerTip = &query.ChainPoint{
		Slot: point.Slot,
		Hash: point.Hash,
	}
	return connect.NewResponse(resp), nil
}

// StreamUtxos
