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

package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"

	_ "go.uber.org/automaxprocs"

	"github.com/blinklabs-io/cardano-node-api/internal/api"
	"github.com/blinklabs-io/cardano-node-api/internal/config"
	"github.com/blinklabs-io/cardano-node-api/internal/logging"
	"github.com/blinklabs-io/cardano-node-api/internal/node"
	"github.com/blinklabs-io/cardano-node-api/internal/utxorpc"
	"github.com/blinklabs-io/cardano-node-api/internal/version"
)

var cmdlineFlags struct {
	configFile string
}

func main() {
	flag.StringVar(
		&cmdlineFlags.configFile,
		"config",
		"",
		"path to config file to load",
	)
	flag.Parse()

	// Load config
	cfg, err := config.Load(cmdlineFlags.configFile)
	if err != nil {
		fmt.Printf("Failed to load config: %s\n", err)
		os.Exit(1)
	}

	// Configure logging
	logging.Configure()
	logger := logging.GetLogger()

	// Test node connection
	if cfg.Node.SkipCheck {
		logger.Debug("skipping node check")
	} else {
		if oConn, err := node.GetConnection(nil); err != nil {
			logger.Error("failed to connect to node:", "error", err)
		} else {
			oConn.Close()
		}
	}

	logger.Info(
		"starting cardano-node-api version",
		"version",
		version.GetVersionString(),
	)

	// Start debug listener
	if cfg.Debug.ListenPort > 0 {
		logger.Info(fmt.Sprintf(
			"starting debug listener on %s:%d",
			cfg.Debug.ListenAddress,
			cfg.Debug.ListenPort,
		))
		go func() {
			err := http.ListenAndServe(
				fmt.Sprintf(
					"%s:%d",
					cfg.Debug.ListenAddress,
					cfg.Debug.ListenPort,
				),
				nil,
			)
			if err != nil {
				logger.Error("failed to start debug listener:", "error", err)
				os.Exit(1)
			}
		}()
	}

	// Start API listener
	go func() {
		if err := api.Start(cfg); err != nil {
			logger.Error("failed to start API:", "error", err)
			os.Exit(1)
		}
	}()

	// Start UTxO RPC gRPC listener
	if err := utxorpc.Start(cfg); err != nil {
		logger.Error("failed to start gRPC:", "error", err)
		os.Exit(1)
	}

	// Wait forever
	select {}
}
