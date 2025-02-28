// Package logger provides a simple logger for terragrunt-ls.
package logger

import (
	"io"
	"log/slog"
	"os"
)

// Logger is a wrapper around slog.Logger that provides additional methods
type Logger struct {
	*slog.Logger
	closer io.Closer
}

// NewLogger builds the standard logger for terragrunt-ls.
//
// When supplied with a filename, it'll create a new file and write logs to it.
// Otherwise, it'll write logs to stderr.
func NewLogger(filename string) *Logger {
	var (
		logWriter io.Writer
		closer    io.Closer
	)

	if filename != "" {
		const readWritePerm = 0666

		file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, readWritePerm)
		if err != nil {
			slog.Error("Failed to open log file", "error", err)
			os.Exit(1)
		}

		logWriter = file
		closer = file
	} else {
		logWriter = os.Stderr
		closer = nil
	}

	// Create a JSON handler for structured logging
	handler := slog.NewJSONHandler(logWriter, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	logger := slog.New(handler)

	return &Logger{
		Logger: logger,
		closer: closer,
	}
}

// Close closes the logger
func (l *Logger) Close() error {
	if l.closer != nil {
		return l.closer.Close()
	}

	return nil
}

// Debug logs a debug message
func (l *Logger) Debug(args ...interface{}) {
	l.Logger.Debug("debug", "msg", args)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Logger.Debug(format, args...)
}

// Info logs an info message
func (l *Logger) Info(args ...interface{}) {
	l.Logger.Info("info", "msg", args)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Logger.Info(format, args...)
}

// Error logs an error message
func (l *Logger) Error(args ...interface{}) {
	l.Logger.Error("error", "msg", args)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Logger.Error(format, args...)
}
