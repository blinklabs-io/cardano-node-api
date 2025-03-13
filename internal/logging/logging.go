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

package logging

import (
	"log/slog"
	"os"
	"time"

	"github.com/blinklabs-io/cardano-node-api/internal/config"
)

var (
	globalLogger *slog.Logger
	accessLogger *slog.Logger
)

// Configure initializes the global logger.
func Configure() {
	cfg := config.GetConfig()
	var level slog.Level
	switch cfg.Logging.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.String(
					"timestamp",
					a.Value.Time().Format(time.RFC3339),
				)
			}
			return a
		},
		Level: level,
	})
	globalLogger = slog.New(handler).With("component", "main")

	// Configure access logger
	accessHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.String(
					"timestamp",
					a.Value.Time().Format(time.RFC3339),
				)
			}
			return a
		},
		Level: slog.LevelInfo,
	})
	accessLogger = slog.New(accessHandler).With("component", "access")
}

// GetLogger returns the global application logger.
func GetLogger() *slog.Logger {
	if globalLogger == nil {
		Configure()
	}
	return globalLogger
}

// GetAccessLogger returns the access logger.
func GetAccessLogger() *slog.Logger {
	if accessLogger == nil {
		Configure()
	}
	return accessLogger
}
