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
	"errors"
	"fmt"
	"log"

	connect "connectrpc.com/connect"
	"github.com/blinklabs-io/adder/event"
	input_chainsync "github.com/blinklabs-io/adder/input/chainsync"
	"github.com/blinklabs-io/cardano-node-api/internal/node"
	ocommon "github.com/blinklabs-io/gouroboros/protocol/common"
	watch "github.com/utxorpc/go-codegen/utxorpc/v1alpha/watch"
	"github.com/utxorpc/go-codegen/utxorpc/v1alpha/watch/watchconnect"
)

// watchServiceServer implements the WatchService API
type watchServiceServer struct {
	watchconnect.UnimplementedWatchServiceHandler
}

// WatchTx
func (s *watchServiceServer) WatchTx(
	ctx context.Context,
	req *connect.Request[watch.WatchTxRequest],
	stream *connect.ServerStream[watch.WatchTxResponse],
) error {
	predicate := req.Msg.GetPredicate() // Predicate
	fieldMask := req.Msg.GetFieldMask()
	log.Printf(
		"Got a WatchTx request with predicate %v and fieldMask %v",
		predicate,
		fieldMask,
	)

	// Setup event channel
	eventChan := make(chan event.Event, 10)
	connCfg := node.ConnectionConfig{
		ChainSyncEventChan: eventChan,
	}
	// Connect to node
	oConn, err := node.GetConnection(&connCfg)
	if err != nil {
		log.Printf("ERROR: %s", err)
		return err
	}
	defer func() {
		// Close Ouroboros connection
		oConn.Close()
	}()

	// Get current tip
	tip, err := oConn.ChainSync().Client.GetCurrentTip()
	if err != nil {
		log.Printf("ERROR: %s", err)
		return err
	}
	// Start the sync with the node
	err = oConn.ChainSync().Client.Sync([]ocommon.Point{tip.Point})
	if err != nil {
		log.Printf("ERROR: %s", err)
		return err
	}

	// Wait for events
	for {
		evt, ok := <-eventChan
		if !ok {
			log.Printf("ERROR: channel closed")
			return errors.New("ERROR: channel closed")
		}

		switch evt.Type {
		case "chainsync.block":
			// Get event context to get the block chain information
			context := evt.Context
			if context == nil {
				log.Printf("ERROR: empty block context")
				return errors.New("ERROR: empty block context")
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
			block := be.Block // gOuorboros Block

			// Loop through transactions
			for _, tx := range block.Transactions() {
				resp := &watch.WatchTxResponse{}
				var act watch.AnyChainTx
				var actc watch.AnyChainTx_Cardano
				cTx := tx.Utxorpc() // *cardano.Tx
				actc.Cardano = cTx
				act.Chain = &actc
				var wtra watch.WatchTxResponse_Apply
				wtra.Apply = &act
				resp.Action = &wtra
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
		// Log event
		log.Printf("event: %v", evt)
	}
}
