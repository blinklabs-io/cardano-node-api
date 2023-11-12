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
	"github.com/blinklabs-io/cardano-node-api/internal/api"
	"github.com/blinklabs-io/cardano-node-api/internal/config"
	"github.com/blinklabs-io/cardano-node-api/internal/logging"
	"github.com/blinklabs-io/cardano-node-api/internal/node"
	"net/http"
	_ "net/http/pprof"
	"os"
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
	logging.Setup(&cfg.Logging)
	logger := logging.GetLogger()
	// Sync logger on exit
	defer func() {
		if err := logger.Sync(); err != nil {
			// We don't actually care about the error here, but we have to do something
			// to appease the linter
			return
		}
	}()

	// Test node connection
	if oConn, err := node.GetConnection(); err != nil {
		logger.Fatalf("failed to connect to node: %s", err)
	} else {
		oConn.Close()
	}

	// Start debug listener
	if cfg.Debug.ListenPort > 0 {
		logger.Infof(
			"starting debug listener on %s:%d",
			cfg.Debug.ListenAddress,
			cfg.Debug.ListenPort,
		)
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
				logger.Fatalf("failed to start debug listener: %s", err)
			}
		}()
	}

	// Start API listener
	logger.Infof(
		"starting API listener on %s:%d",
		cfg.Api.ListenAddress,
		cfg.Api.ListenPort,
	)
	if err := api.Start(cfg); err != nil {
		logger.Fatalf("failed to start API: %s", err)
	}

	// Wait forever
	select {}
}
