package logging

import (
	"github.com/blinklabs-io/cardano-node-api/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"time"
)

type Logger = zap.SugaredLogger

var globalLogger *Logger

func Setup(cfg *config.LoggingConfig) {
	// Build our custom logging config
	loggerConfig := zap.NewProductionConfig()
	// Change timestamp key name
	loggerConfig.EncoderConfig.TimeKey = "timestamp"
	// Use a human readable time format
	loggerConfig.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(
		time.RFC3339,
	)

	// Set level
	if cfg.Level != "" {
		level, err := zapcore.ParseLevel(cfg.Level)
		if err != nil {
			log.Fatalf("error configuring logger: %s", err)
		}
		loggerConfig.Level.SetLevel(level)
	}

	// Create the logger
	l, err := loggerConfig.Build()
	if err != nil {
		log.Fatal(err)
	}

	// Store the "sugared" version of the logger
	globalLogger = l.Sugar()
}

func GetLogger() *zap.SugaredLogger {
	return globalLogger
}

func GetDesugaredLogger() *zap.Logger {
	return globalLogger.Desugar()
}

func GetAccessLogger() *zap.Logger {
	return globalLogger.Desugar().
		With(zap.String("type", "access")).
		WithOptions(zap.WithCaller(false))
}
