// Package testutils provides utilities for testing.
package testutils

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"terragrunt-ls/internal/logger"
	"testing"
)

var _ logger.Logger = &testLogger{}

// Logger is a wrapper around slog.Logger that provides additional methods
type testLogger struct {
	*slog.Logger
	closer io.Closer
}

func NewTestLogger(t *testing.T) *testLogger {
	t.Helper()

	// Create a test logger that writes to the test log
	testWriter := testWriter{t}
	handler := slog.NewJSONHandler(testWriter, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	slogger := slog.New(handler)

	// Create a new logger with the test writer
	return &testLogger{
		Logger: slogger,
	}
}

// testWriter implements io.Writer and writes to the test log
type testWriter struct {
	t *testing.T
}

func (tw testWriter) Write(p []byte) (n int, err error) {
	tw.t.Log(string(p))
	return len(p), nil
}

func PointerOfInt(i int) *int {
	return &i
}

func CreateFile(dir, name, content string) (string, error) {
	const ownerRWGlobalR = 0644

	return CreateFileWithMode(dir, name, content, ownerRWGlobalR)
}

func CreateFileWithMode(dir, name, content string, mode os.FileMode) (string, error) {
	path := filepath.Join(dir, name)

	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		return "", err
	}

	return path, nil
}

// Close closes the logger
func (l *testLogger) Close() error {
	if l.closer != nil {
		return l.closer.Close()
	}

	return nil
}

// Debug logs a debug message
func (l *testLogger) Debug(msg string, args ...interface{}) {
	l.Logger.Debug(msg, args...)
}

// Info logs an info message
func (l *testLogger) Info(msg string, args ...interface{}) {
	l.Logger.Info(msg, args...)
}

// Error logs an error message
func (l *testLogger) Error(msg string, args ...interface{}) {
	l.Logger.Error(msg, args...)
}
