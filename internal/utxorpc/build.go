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
	"log"

	connect "connectrpc.com/connect"
	build "github.com/utxorpc/go-codegen/utxorpc/v1alpha/build"
	"github.com/utxorpc/go-codegen/utxorpc/v1alpha/build/buildconnect"

	"github.com/blinklabs-io/cardano-node-api/internal/node"
)

// ledgerStateServiceServer implements the LedgerStateService API
type ledgerStateServiceServer struct {
	buildconnect.UnimplementedLedgerStateServiceHandler
}

// GetChainTip
func (s *ledgerStateServiceServer) GetChainTip(
	ctx context.Context,
	req *connect.Request[build.GetChainTipRequest],
) (*connect.Response[build.GetChainTipResponse], error) {
	log.Printf("Got a GetChainTip request")

	// Connect to node
	oConn, err := node.GetConnection()
	if err != nil {
		return nil, err
	}
	defer func() {
		// Close Ouroboros connection
		oConn.Close()
	}()

	resp := &build.GetChainTipResponse{}
	// Start client
	tip, err := oConn.ChainSync().Client.GetCurrentTip()
	if err != nil {
		return nil, err
	}
	point := tip.Point
	resp.Tip = &build.ChainPoint{
		Slot:   point.Slot,
		Height: tip.BlockNumber,
		Hash:   point.Hash,
	}
	return connect.NewResponse(resp), nil
}

// TODO: complete these

// GetChainParam
func (s *ledgerStateServiceServer) GetChainParam(
	ctx context.Context,
	req *connect.Request[build.GetChainParamRequest],
) (*connect.Response[build.GetChainParamResponse], error) {
	return nil, nil
}

// GetUtxoByAddress
func (s *ledgerStateServiceServer) GetUtxoByAddress(
	ctx context.Context,
	req *connect.Request[build.GetUtxoByAddressRequest],
) (*connect.Response[build.GetUtxoByAddressResponse], error) {
	return nil, nil
}

// GetUtxoByRef
func (s *ledgerStateServiceServer) GetUtxoByRef(
	ctx context.Context,
	req *connect.Request[build.GetUtxoByRefRequest],
) (*connect.Response[build.GetUtxoByRefResponse], error) {
	return nil, nil
}

// HoldUtxo
func (s *ledgerStateServiceServer) HoldUtxo(
	ctx context.Context,
	req *connect.Request[build.HoldUtxoRequest],
	stream *connect.ServerStream[build.HoldUtxoResponse],
) error {
	return nil
}
