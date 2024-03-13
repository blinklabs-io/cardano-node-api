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

package node

import (
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/protocol/chainsync"
	"github.com/blinklabs-io/gouroboros/protocol/common"

	"github.com/blinklabs-io/cardano-node-api/internal/config"
)

func buildChainSyncConfig() chainsync.Config {
	cfg := config.GetConfig()
	return chainsync.NewConfig(
		chainsync.WithBlockTimeout(
			time.Duration(cfg.Node.Timeout) * time.Second,
		),
		chainsync.WithRollBackwardFunc(chainSyncRollBackwardHandler),
		chainsync.WithRollForwardFunc(chainSyncRollForwardHandler),
	)
}

func chainSyncRollBackwardHandler(point common.Point, tip chainsync.Tip) error {
	log.Printf("roll backward: point = %#v, tip = %#v\n", point, tip)
	return nil
}

func chainSyncRollForwardHandler(
	blockType uint,
	blockData interface{},
	tip chainsync.Tip,
) error {
	var block ledger.Block
	switch v := blockData.(type) {
	case ledger.Block:
		block = v
	case ledger.BlockHeader:
		blockSlot := v.SlotNumber()
		blockHash, _ := hex.DecodeString(v.Hash())
		oConn, err := GetConnection()
		if err != nil {
			return err
		}
		block, err = oConn.BlockFetch().Client.GetBlock(common.NewPoint(blockSlot, blockHash))
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown block data")
	}
	// Display block info
	switch blockType {
	case ledger.BlockTypeByronEbb:
		byronEbbBlock := block.(*ledger.ByronEpochBoundaryBlock)
		log.Printf(
			"era = Byron (EBB), epoch = %d, slot = %d, id = %s\n",
			byronEbbBlock.Header.ConsensusData.Epoch,
			byronEbbBlock.SlotNumber(),
			byronEbbBlock.Hash(),
		)
	case ledger.BlockTypeByronMain:
		byronBlock := block.(*ledger.ByronMainBlock)
		log.Printf(
			"era = Byron, epoch = %d, slot = %d, id = %s\n",
			byronBlock.Header.ConsensusData.SlotId.Epoch,
			byronBlock.SlotNumber(),
			byronBlock.Hash(),
		)
	default:
		log.Printf(
			"era = %s, slot = %d, block_no = %d, id = %s\n",
			block.Era().Name,
			block.SlotNumber(),
			block.BlockNumber(),
			block.Hash(),
		)
	}
	return nil
}
