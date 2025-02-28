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

package node

import (
	"errors"
	"time"

	"github.com/blinklabs-io/cardano-node-api/internal/config"

	"github.com/blinklabs-io/adder/event"
	input_chainsync "github.com/blinklabs-io/adder/input/chainsync"
	"github.com/blinklabs-io/gouroboros/ledger"
	"github.com/blinklabs-io/gouroboros/protocol/chainsync"
	"github.com/blinklabs-io/gouroboros/protocol/common"
)

func buildChainSyncConfig(connCfg ConnectionConfig) chainsync.Config {
	cfg := config.GetConfig()
	// #nosec G115
	return chainsync.NewConfig(
		chainsync.WithBlockTimeout(
			time.Duration(cfg.Node.QueryTimeout)*time.Second,
		),
		// We wrap the handler funcs to include our ConnectionConfig
		chainsync.WithRollBackwardFunc(
			func(connCfg ConnectionConfig) chainsync.RollBackwardFunc {
				return func(ctx chainsync.CallbackContext, point common.Point, tip chainsync.Tip) error {
					return chainSyncRollBackwardHandler(
						ctx, connCfg, point, tip,
					)
				}
			}(connCfg),
		),
		chainsync.WithRollForwardFunc(
			func(connCfg ConnectionConfig) chainsync.RollForwardFunc {
				return func(ctx chainsync.CallbackContext, blockType uint, blockData any, tip chainsync.Tip) error {
					return chainSyncRollForwardHandler(
						ctx, connCfg, blockType, blockData, tip,
					)
				}
			}(connCfg),
		),
	)
}

func chainSyncRollBackwardHandler(
	ctx chainsync.CallbackContext,
	connCfg ConnectionConfig,
	point common.Point,
	tip chainsync.Tip,
) error {
	if connCfg.ChainSyncEventChan != nil {
		evt := event.New(
			"chainsync.rollback",
			time.Now(),
			nil,
			input_chainsync.NewRollbackEvent(point),
		)
		connCfg.ChainSyncEventChan <- evt
	}
	return nil
}

func chainSyncRollForwardHandler(
	ctx chainsync.CallbackContext,
	connCfg ConnectionConfig,
	blockType uint,
	blockData interface{},
	tip chainsync.Tip,
) error {
	cfg := config.GetConfig()
	if connCfg.ChainSyncEventChan != nil {
		switch v := blockData.(type) {
		case ledger.Block:
			// Emit block-level event
			blockEvt := event.New(
				"chainsync.block",
				time.Now(),
				input_chainsync.NewBlockContext(v, cfg.Node.NetworkMagic),
				input_chainsync.NewBlockEvent(v, true),
			)
			connCfg.ChainSyncEventChan <- blockEvt
			// Emit transaction-level events
			for t, transaction := range v.Transactions() {
				// TODO: do we need to resolve inputs?
				// resolvedInputs, err := resolveTransactionInputs(transaction, connCfg)
				// if err != nil {
				// 	return fmt.Errorf("failed to resolve inputs for transaction: %w", err)
				// }
				txEvt := event.New(
					"chainsync.transaction",
					time.Now(),
					// #nosec G115
					input_chainsync.NewTransactionContext(v, transaction, uint32(t), cfg.Node.NetworkMagic),
					input_chainsync.NewTransactionEvent(v, transaction, true, nil),
				)
				connCfg.ChainSyncEventChan <- txEvt
			}
		/*
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
		*/
		default:
			return errors.New("unknown block data")
		}
	}
	return nil
}
