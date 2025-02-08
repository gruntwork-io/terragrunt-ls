// Package logger provides a simple logger for terragrunt-ls.
package logger

import (
	"log"

	"go.uber.org/zap"
)

// NewLogger builds the standard logger for terragrunt-ls.
//
// When supplied with a filename, it'll create a new file and write logs to it.
// Otherwise, it'll write logs to stderr.
func NewLogger(filename string) *zap.Logger {
	if filename != "" {
		config := zap.NewDevelopmentConfig()
		config.OutputPaths = []string{filename}

		logger, err := config.Build()
		if err != nil {
			log.Fatal(err)
		}

		return logger
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}

	return logger
}
