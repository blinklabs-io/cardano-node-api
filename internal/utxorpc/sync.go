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
	"log"

	connect "connectrpc.com/connect"
	ocommon "github.com/blinklabs-io/gouroboros/protocol/common"
	sync "github.com/utxorpc/go-codegen/utxorpc/v1alpha/sync"
	"github.com/utxorpc/go-codegen/utxorpc/v1alpha/sync/syncconnect"

	"github.com/blinklabs-io/cardano-node-api/internal/node"
)

// chainSyncServiceServer implements the ChainSyncService API
type chainSyncServiceServer struct {
	syncconnect.UnimplementedChainSyncServiceHandler
}

// FetchBlock
func (s *chainSyncServiceServer) FetchBlock(
	ctx context.Context,
	req *connect.Request[sync.FetchBlockRequest],
) (*connect.Response[sync.FetchBlockResponse], error) {
	ref := req.Msg.GetRef() // BlockRef
	fieldMask := req.Msg.GetFieldMask()
	log.Printf("Got a FetchBlock request with ref %v and fieldMask %v", ref, fieldMask)

	// Connect to node
	oConn, err := node.GetConnection(true)
	if err != nil {
		return nil, err
	}
	// Async error handler
	go func() {
		_, ok := <-oConn.ErrorChan()
		if !ok {
			return
		}
	}()
	defer func() {
		// Close Ouroboros connection
		oConn.Close()
	}()

	resp := &sync.FetchBlockResponse{}
	// Start client
	var points []ocommon.Point
	if len(ref) > 0 {
		for _, blockRef := range ref {
			blockIdx := blockRef.GetIndex()
			blockHash := blockRef.GetHash()
			hash, _ := hex.DecodeString(string(blockHash))
			slot := uint64(blockIdx)
			point := ocommon.NewPoint(slot, hash)
			points = append(points, point)
		}
	} else {
		tip, err := oConn.ChainSync().Client.GetCurrentTip()
		if err != nil {
			return nil, err
		}
		point := tip.Point
		points = append(points, point)
	}
	for _, point := range points {
		log.Printf("Point Slot: %d, Hash: %x\n", point.Slot, point.Hash)
		block, err := oConn.BlockFetch().Client.GetBlock(
			ocommon.NewPoint(point.Slot, point.Hash),
		)
		if err != nil {
			return nil, err
		}
		var acb sync.AnyChainBlock
		var acbc sync.AnyChainBlock_Cardano
		ret := NewBlockFromBlock(block)
		acbc.Cardano = &ret
		acb.Chain = &acbc
		resp.Block = append(resp.Block, &acb)
	}

	return connect.NewResponse(resp), nil
}

// DumpHistory
func (s *chainSyncServiceServer) DumpHistory(
	ctx context.Context,
	req *connect.Request[sync.DumpHistoryRequest],
) (*connect.Response[sync.DumpHistoryResponse], error) {
	startToken := req.Msg.GetStartToken() // BlockRef
	maxItems := req.Msg.GetMaxItems()
	fieldMask := req.Msg.GetFieldMask()
	log.Printf("Got a DumpHistory request with token %v and maxItems %d and fieldMask %v", startToken, maxItems, fieldMask)

	// Connect to node
	oConn, err := node.GetConnection(true)
	if err != nil {
		return nil, err
	}
	// Async error handler
	go func() {
		_, ok := <-oConn.ErrorChan()
		if !ok {
			return
		}
	}()
	defer func() {
		// Close Ouroboros connection
		oConn.Close()
	}()

	resp := &sync.DumpHistoryResponse{}
	// Start client
	log.Printf("startToken: %#v\n", startToken)
	var startPoint ocommon.Point
	if startToken != nil {
		log.Printf("startToken != nil\n")
		blockRef := startToken
		blockIdx := blockRef.GetIndex()
		blockHash := blockRef.GetHash()
		hash, _ := hex.DecodeString(string(blockHash))
		slot := uint64(blockIdx)
		startPoint = ocommon.NewPoint(slot, hash)
	} else {
		log.Printf("getting tip\n")
		tip, err := oConn.ChainSync().Client.GetCurrentTip()
		if err != nil {
			return nil, err
		}
		startPoint = tip.Point
	}
	log.Printf("startPoint slot %d, hash %x\n", startPoint.Slot, startPoint.Hash)
	// TODO: why is this giving us 0?
	start, end, err := oConn.ChainSync().Client.GetAvailableBlockRange(
		[]ocommon.Point{startPoint},
	)
	log.Printf("Start:     slot %d, hash %x\n", start.Slot, start.Hash)
	log.Printf("End (tip): slot %d, hash %x\n", end.Slot, end.Hash)

	return connect.NewResponse(resp), nil
}

// FollowTip
func (s *chainSyncServiceServer) FollowTip(
	ctx context.Context,
	req *connect.Request[sync.FollowTipRequest],
	stream *connect.ServerStream[sync.FollowTipResponse],
) error {
	intersect := req.Msg.GetIntersect()
	log.Printf("Got a FollowTip request with intersect %v", intersect)
	// Connect to node
	oConn, err := node.GetConnection(true)
	if err != nil {
		return err
	}
	// Async error handler
	go func() {
		_, ok := <-oConn.ErrorChan()
		if !ok {
			return
		}
	}()
	defer func() {
		// Close Ouroboros connection
		oConn.Close()
	}()

	var point ocommon.Point
	if len(intersect) > 0 {
		for _, blockRef := range intersect {
			blockIdx := blockRef.GetIndex()
			blockHash := blockRef.GetHash()
			log.Printf("BlockRef: idx: %d, hash: %x", blockIdx, blockHash)
			hash, _ := hex.DecodeString(string(blockHash))
			slot := uint64(blockIdx)
			point = ocommon.NewPoint(slot, hash)
		}
	} else {
		tip, _ := oConn.ChainSync().Client.GetCurrentTip()
		point = tip.Point
	}
	log.Printf("DEBUG: point: %#v\n\n", point)
	// TODO: get data / send to stream
	// stream.send(&sync.FollowTipResponse{})
	return nil
}

