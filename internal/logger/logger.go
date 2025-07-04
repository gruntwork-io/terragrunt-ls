// Package logger provides a simple logger for terragrunt-ls.
package logger

import (
	"io"
	"log/slog"
	"os"
)

var _ Logger = &slogLogger{}

// slogLogger is a wrapper around slog.Logger that provides additional methods
type slogLogger struct {
	*slog.Logger
	writer io.WriteCloser
	level  slog.Level
}

type Logger interface {
	Close() error
	Writer() io.WriteCloser
	Level() slog.Level
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// NewLogger builds the standard logger for terragrunt-ls.
//
// When supplied with a filename, it'll create a new file and write logs to it.
// Otherwise, it'll write logs to stderr.
func NewLogger(filename string, level slog.Level) *slogLogger {
	if filename == "" {
		handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: level,
		})
		logger := slog.New(handler)

		return &slogLogger{
			Logger: logger,
			level:  level,
		}
	}

	const readWritePerm = 0666

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, readWritePerm)
	if err != nil {
		slog.Error("Failed to open log file", "error", err)
		os.Exit(1)
	}

	handler := slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: level,
	})
	logger := slog.New(handler)

	return &slogLogger{
		Logger: logger,
		writer: file,
		level:  level,
	}
}

// Close closes the logger
func (l *slogLogger) Close() error {
	if l.writer != nil {
		return l.writer.Close()
	}

	return nil
}

// Writer returns the writer for the logger
func (l *slogLogger) Writer() io.WriteCloser {
	return l.writer
}

// Level returns the level of the logger
func (l *slogLogger) Level() slog.Level {
	return l.level
}

// Debug logs a debug message
func (l *slogLogger) Debug(msg string, args ...any) {
	l.Logger.Debug(msg, args...)
}

// Info logs an info message
func (l *slogLogger) Info(msg string, args ...any) {
	l.Logger.Info(msg, args...)
}

// Warn logs a warning message
func (l *slogLogger) Warn(msg string, args ...any) {
	l.Logger.Warn(msg, args...)
}

// Error logs an error message
func (l *slogLogger) Error(msg string, args ...any) {
	l.Logger.Error(msg, args...)
}
