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
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/grpchealth"
	"connectrpc.com/grpcreflect"
	"github.com/blinklabs-io/cardano-node-api/internal/config"
	"github.com/blinklabs-io/cardano-node-api/internal/logging"
	"github.com/utxorpc/go-codegen/utxorpc/v1alpha/query/queryconnect"
	"github.com/utxorpc/go-codegen/utxorpc/v1alpha/submit/submitconnect"
	"github.com/utxorpc/go-codegen/utxorpc/v1alpha/sync/syncconnect"
	"github.com/utxorpc/go-codegen/utxorpc/v1alpha/watch/watchconnect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func Start(cfg *config.Config) error {
	// Standard logging
	logger := logging.GetLogger()
	if cfg.Tls.CertFilePath != "" && cfg.Tls.KeyFilePath != "" {
		logger.Info(fmt.Sprintf(
			"starting gRPC TLS listener on: %s:%d",
			cfg.Utxorpc.ListenAddress,
			cfg.Utxorpc.ListenPort,
		))
	} else {
		logger.Info(fmt.Sprintf(
			"starting gRPC listener on %s:%d",
			cfg.Utxorpc.ListenAddress,
			cfg.Utxorpc.ListenPort,
		))
	}
	mux := http.NewServeMux()
	compress1KB := connect.WithCompressMinBytes(1024)
	queryPath, queryHandler := queryconnect.NewQueryServiceHandler(
		&queryServiceServer{},
		compress1KB,
	)
	submitPath, submitHandler := submitconnect.NewSubmitServiceHandler(
		&submitServiceServer{},
		compress1KB,
	)
	syncPath, syncHandler := syncconnect.NewSyncServiceHandler(
		&chainSyncServiceServer{},
		compress1KB,
	)
	watchPath, watchHandler := watchconnect.NewWatchServiceHandler(
		&watchServiceServer{},
		compress1KB,
	)
	mux.Handle(queryPath, queryHandler)
	mux.Handle(submitPath, submitHandler)
	mux.Handle(syncPath, syncHandler)
	mux.Handle(watchPath, watchHandler)
	mux.Handle(
		grpchealth.NewHandler(
			grpchealth.NewStaticChecker(
				queryconnect.QueryServiceName,
				submitconnect.SubmitServiceName,
				syncconnect.SyncServiceName,
				watchconnect.WatchServiceName,
			),
			compress1KB,
		),
	)
	mux.Handle(
		grpcreflect.NewHandlerV1(
			grpcreflect.NewStaticReflector(
				queryconnect.QueryServiceName,
				submitconnect.SubmitServiceName,
				syncconnect.SyncServiceName,
				watchconnect.WatchServiceName,
			),
			compress1KB,
		),
	)
	mux.Handle(
		grpcreflect.NewHandlerV1Alpha(
			grpcreflect.NewStaticReflector(
				queryconnect.QueryServiceName,
				submitconnect.SubmitServiceName,
				syncconnect.SyncServiceName,
				watchconnect.WatchServiceName,
			),
			compress1KB,
		),
	)
	if cfg.Tls.CertFilePath != "" && cfg.Tls.KeyFilePath != "" {
		server := &http.Server{
			Addr: fmt.Sprintf(
				"%s:%d",
				cfg.Utxorpc.ListenAddress,
				cfg.Utxorpc.ListenPort,
			),
			Handler:           mux,
			ReadHeaderTimeout: 60 * time.Second,
		}
		return server.ListenAndServeTLS(
			cfg.Tls.CertFilePath,
			cfg.Tls.KeyFilePath,
		)
	} else {
		server := &http.Server{
			Addr: fmt.Sprintf(
				"%s:%d",
				cfg.Utxorpc.ListenAddress,
				cfg.Utxorpc.ListenPort,
			),
			// Use h2c so we can serve HTTP/2 without TLS
			Handler:           h2c.NewHandler(mux, &http2.Server{}),
			ReadHeaderTimeout: 60 * time.Second,
		}
		return server.ListenAndServe()
	}
}
