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
	"log"

	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/protocol/blockfetch"
)

func buildBlockFetchConfig() blockfetch.Config {
	return blockfetch.NewConfig(
		blockfetch.WithBlockFunc(blockFetchBlockHandler),
	)
}

func blockFetchBlockHandler(blockData ledger.Block) error {
	switch block := blockData.(type) {
	case *ledger.ByronEpochBoundaryBlock, *ledger.ByronMainBlock:
		log.Printf("my brother in Christ, this is Byron... run...")
	case ledger.Block:
		log.Printf("era = %s, slot = %d, block_no = %d, id = %s\n", block.Era().Name, block.SlotNumber(), block.BlockNumber(), block.Hash())
	}
	return nil
}


