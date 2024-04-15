// Copyright 2023 Blink Labs Software
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
	"fmt"
	"os"

	"github.com/blinklabs-io/cardano-node-api/internal/config"

	ouroboros "github.com/blinklabs-io/gouroboros"
	"github.com/blinklabs-io/snek/event"
)

type ConnectionConfig struct {
	ChainSyncEventChan chan event.Event
}

func GetConnection(connCfg *ConnectionConfig) (*ouroboros.Connection, error) {
	// Make sure we always have a ConnectionConfig object
	if connCfg == nil {
		connCfg = &ConnectionConfig{}
	}
	cfg := config.GetConfig()
	// Connect to cardano-node
	oConn, err := ouroboros.NewConnection(
		ouroboros.WithNetworkMagic(uint32(cfg.Node.NetworkMagic)),
		ouroboros.WithNodeToNode(false),
		ouroboros.WithKeepAlive(true),
		ouroboros.WithChainSyncConfig(buildChainSyncConfig(*connCfg)),
		ouroboros.WithLocalTxMonitorConfig(buildLocalTxMonitorConfig()),
		ouroboros.WithLocalStateQueryConfig(buildLocalStateQueryConfig()),
		ouroboros.WithLocalTxSubmissionConfig(buildLocalTxSubmissionConfig()),
	)
	if err != nil {
		return nil, fmt.Errorf("failure creating Ouroboros connection: %s", err)
	}

	if cfg.Node.Address != "" && cfg.Node.Port > 0 {
		// Connect to TCP port
		if err := oConn.Dial("tcp", fmt.Sprintf("%s:%d", cfg.Node.Address, cfg.Node.Port)); err != nil {
			return nil, fmt.Errorf(
				"failure connecting to node via TCP: %s",
				err,
			)
		}
	} else if cfg.Node.SocketPath != "" {
		// Check that node socket path exists
		if _, err := os.Stat(cfg.Node.SocketPath); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("node socket path does not exist: %s", cfg.Node.SocketPath)
			} else {
				return nil, fmt.Errorf("unknown error checking if node socket path exists: %s", err)
			}
		}
		if err := oConn.Dial("unix", cfg.Node.SocketPath); err != nil {
			return nil, fmt.Errorf("failure connecting to node via UNIX socket: %s", err)
		}
	} else {
		return nil, fmt.Errorf("you must specify either the UNIX socket path or the address/port for your cardano-node")
	}
	return oConn, nil
}
