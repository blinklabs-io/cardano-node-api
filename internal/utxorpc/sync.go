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
	"fmt"
	"log"

	connect "connectrpc.com/connect"
	"github.com/blinklabs-io/adder/event"
	input_chainsync "github.com/blinklabs-io/adder/input/chainsync"
	ocommon "github.com/blinklabs-io/gouroboros/protocol/common"
	sync "github.com/utxorpc/go-codegen/utxorpc/v1alpha/sync"
	"github.com/utxorpc/go-codegen/utxorpc/v1alpha/sync/syncconnect"

	"github.com/blinklabs-io/cardano-node-api/internal/node"
)

// chainSyncServiceServer implements the ChainSyncService API
type chainSyncServiceServer struct {
	syncconnect.UnimplementedSyncServiceHandler
}

// FetchBlock
func (s *chainSyncServiceServer) FetchBlock(
	ctx context.Context,
	req *connect.Request[sync.FetchBlockRequest],
) (*connect.Response[sync.FetchBlockResponse], error) {
	ref := req.Msg.GetRef() // BlockRef
	fieldMask := req.Msg.GetFieldMask()
	log.Printf(
		"Got a FetchBlock request with ref %v and fieldMask %v",
		ref,
		fieldMask,
	)

	// Connect to node
	oConn, err := node.GetConnection(nil)
	if err != nil {
		return nil, err
	}
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
			slot := uint64(blockIdx)
			point := ocommon.NewPoint(slot, blockHash)
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
	log.Printf("points: %v", points)
	// TODO: replace with something that works NtC
	// for _, point := range points {
	// 	log.Printf("Point Slot: %d, Hash: %x\n", point.Slot, point.Hash)
	// 	block, err := oConn.BlockFetch().Client.GetBlock(
	// 		ocommon.NewPoint(point.Slot, point.Hash),
	// 	)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	var acb sync.AnyChainBlock
	// 	var acbc sync.AnyChainBlock_Cardano
	// 	ret := NewBlockFromBlock(block)
	// 	acbc.Cardano = &ret
	// 	acb.Chain = &acbc
	// 	resp.Block = append(resp.Block, &acb)
	// }

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
	log.Printf(
		"Got a DumpHistory request with token %v and maxItems %d and fieldMask %v",
		startToken,
		maxItems,
		fieldMask,
	)

	// Connect to node
	oConn, err := node.GetConnection(nil)
	if err != nil {
		return nil, err
	}
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
		slot := uint64(blockIdx)
		startPoint = ocommon.NewPoint(slot, blockHash)
	} else {
		log.Printf("getting tip\n")
		tip, err := oConn.ChainSync().Client.GetCurrentTip()
		if err != nil {
			return nil, err
		}
		startPoint = tip.Point
	}
	log.Printf(
		"startPoint slot %d, hash %x\n",
		startPoint.Slot,
		startPoint.Hash,
	)
	// TODO: why is this giving us 0?
	start, end, err := oConn.ChainSync().Client.GetAvailableBlockRange(
		[]ocommon.Point{startPoint},
	)
	if err != nil {
		return nil, err
	}
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
	intersect := req.Msg.GetIntersect() // []*BlockRef
	log.Printf("Got a FollowTip request with intersect %v", intersect)

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
	if len(intersect) > 0 {
		for _, blockRef := range intersect {
			blockIdx := blockRef.GetIndex()
			blockHash := blockRef.GetHash()
			log.Printf("BlockRef: idx: %d, hash: %x", blockIdx, blockHash)
			slot := uint64(blockIdx)
			point = ocommon.NewPoint(slot, blockHash)
		}
	} else {
		tip, _ := oConn.ChainSync().Client.GetCurrentTip()
		point = tip.Point
	}

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

		switch evt.Type {
		case "chainsync.block":
			resp := &sync.FollowTipResponse{}
			// Get event context to get the block chain information
			context := evt.Context
			if context == nil {
				log.Printf("ERROR: empty block context")
				return fmt.Errorf("ERROR: empty block context")
			}
			bc := context.(input_chainsync.BlockContext)
			// Get event payload to get the block data
			payload := evt.Payload
			if payload == nil {
				log.Printf(
					"ERROR: empty payload: block: %d, slot: %d",
					bc.BlockNumber,
					bc.SlotNumber,
				)
				return fmt.Errorf(
					"ERROR: empty payload: block: %d, slot: %d",
					bc.BlockNumber,
					bc.SlotNumber,
				)
			}
			be := payload.(input_chainsync.BlockEvent)
			block := be.Block // gOuroboros Block

			var acb sync.AnyChainBlock
			var acbc sync.AnyChainBlock_Cardano
			acbc.Cardano = block.Utxorpc()
			acb.Chain = &acbc
			var ftra sync.FollowTipResponse_Apply
			ftra.Apply = &acb
			resp.Action = &ftra
			err = stream.Send(resp)
			if err != nil {
				return err
			}
			// Log event
			log.Printf(
				"block: slot: %d, hash: %s",
				block.SlotNumber(),
				block.Hash(),
			)
		}
	}
}
