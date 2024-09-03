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

	"connectrpc.com/connect"
	"connectrpc.com/grpchealth"
	"connectrpc.com/grpcreflect"
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
			grpchealth.NewStaticChecker(queryconnect.QueryServiceName),
			compress1KB,
		),
	)
	mux.Handle(
		grpchealth.NewHandler(
			grpchealth.NewStaticChecker(submitconnect.SubmitServiceName),
			compress1KB,
		),
	)
	mux.Handle(
		grpchealth.NewHandler(
			grpchealth.NewStaticChecker(syncconnect.SyncServiceName),
			compress1KB,
		),
	)
	mux.Handle(
		grpchealth.NewHandler(
			grpchealth.NewStaticChecker(watchconnect.WatchServiceName),
			compress1KB,
		),
	)
	mux.Handle(
		grpcreflect.NewHandlerV1(
			grpcreflect.NewStaticReflector(queryconnect.QueryServiceName),
			compress1KB,
		),
	)
	mux.Handle(
		grpcreflect.NewHandlerV1(
			grpcreflect.NewStaticReflector(submitconnect.SubmitServiceName),
			compress1KB,
		),
	)
	mux.Handle(
		grpcreflect.NewHandlerV1(
			grpcreflect.NewStaticReflector(syncconnect.SyncServiceName),
			compress1KB,
		),
	)
	mux.Handle(
		grpcreflect.NewHandlerV1(
			grpcreflect.NewStaticReflector(watchconnect.WatchServiceName),
			compress1KB,
		),
	)
	mux.Handle(
		grpcreflect.NewHandlerV1Alpha(
			grpcreflect.NewStaticReflector(queryconnect.QueryServiceName),
			compress1KB,
		),
	)
	mux.Handle(
		grpcreflect.NewHandlerV1Alpha(
			grpcreflect.NewStaticReflector(submitconnect.SubmitServiceName),
			compress1KB,
		),
	)
	mux.Handle(
		grpcreflect.NewHandlerV1Alpha(
			grpcreflect.NewStaticReflector(syncconnect.SyncServiceName),
			compress1KB,
		),
	)
	mux.Handle(
		grpcreflect.NewHandlerV1Alpha(
			grpcreflect.NewStaticReflector(watchconnect.WatchServiceName),
			compress1KB,
		),
	)
	if cfg.Tls.CertFilePath != "" && cfg.Tls.KeyFilePath != "" {
		err := http.ListenAndServeTLS(
			fmt.Sprintf(
				"%s:%d",
				cfg.Utxorpc.ListenAddress,
				cfg.Utxorpc.ListenPort,
			),
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
