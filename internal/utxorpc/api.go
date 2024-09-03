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
	"fmt"
	"net/http"

	"github.com/utxorpc/go-codegen/utxorpc/v1alpha/query/queryconnect"
	"github.com/utxorpc/go-codegen/utxorpc/v1alpha/submit/submitconnect"
	"github.com/utxorpc/go-codegen/utxorpc/v1alpha/sync/syncconnect"
	"github.com/utxorpc/go-codegen/utxorpc/v1alpha/watch/watchconnect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/blinklabs-io/cardano-node-api/internal/config"
)

func Start(cfg *config.Config) error {
	mux := http.NewServeMux()
	queryPath, queryHandler := queryconnect.NewQueryServiceHandler(
		&queryServiceServer{},
	)
	submitPath, submitHandler := submitconnect.NewSubmitServiceHandler(
		&submitServiceServer{},
	)
	syncPath, syncHandler := syncconnect.NewSyncServiceHandler(
		&chainSyncServiceServer{},
	)
	watchPath, watchHandler := watchconnect.NewWatchServiceHandler(
		&watchServiceServer{},
	)
	mux.Handle(queryPath, queryHandler)
	mux.Handle(submitPath, submitHandler)
	mux.Handle(syncPath, syncHandler)
	mux.Handle(watchPath, watchHandler)
	if cfg.Tls.CertFilePath != "" && cfg.Tls.KeyFilePath != "" {
		err := http.ListenAndServeTLS(fmt.Sprintf("%s:%d", cfg.Utxorpc.ListenAddress, cfg.Utxorpc.ListenPort),
			cfg.Tls.CertFilePath,
			cfg.Tls.KeyFilePath,
			nil,
		)
		return err
	} else {
		err := http.ListenAndServe(
			fmt.Sprintf("%s:%d", cfg.Utxorpc.ListenAddress, cfg.Utxorpc.ListenPort),
			// Use h2c so we can serve HTTP/2 without TLS
			h2c.NewHandler(mux, &http2.Server{}),
		)
		return err
	}
}
