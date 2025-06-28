// Package config provides configuration loading for terragrunt-ls.
package config

import (
	"log/slog"
	"os"
)

// Config holds the configuration for terragrunt-ls
type Config struct {
	// LogFile is the path to the log file, empty string means stderr
	LogFile string
	// LogLevel is the log level to use
	LogLevel slog.Level
}

const (
	// EnvLogFile is the environment variable that specifies the log file.
	EnvLogFile = "TG_LS_LOG"
	// EnvLogLevel is the environment variable that specifies the log level.
	EnvLogLevel = "TG_LS_LOG_LEVEL"
)

// Load reads configuration from environment variables and returns a populated Config
func Load() *Config {
	cfg := &Config{
		LogFile:  os.Getenv(EnvLogFile),
		LogLevel: slog.LevelInfo, // default level
	}

	// Parse log level from environment variable
	if envLevel := os.Getenv(EnvLogLevel); envLevel != "" {
		levelVar := slog.LevelVar{}
		if err := levelVar.UnmarshalText([]byte(envLevel)); err != nil {
			slog.Error("Failed to parse log level", "error", err)
		} else {
			cfg.LogLevel = levelVar.Level()
		}
	}

	return cfg
}
