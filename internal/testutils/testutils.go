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
	writer io.WriteCloser
	level  slog.Level
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
		writer: testWriter,
		level:  slog.LevelDebug,
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

func (tw testWriter) Close() error {
	return nil
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
	if l.writer != nil {
		return l.writer.Close()
	}

	return nil
}

// Writer returns the writer for the logger
func (l *testLogger) Writer() io.WriteCloser {
	return l.writer
}

// Level returns the level of the logger
func (l *testLogger) Level() slog.Level {
	return l.level
}

// Debug logs a debug message
func (l *testLogger) Debug(msg string, args ...any) {
	l.Logger.Debug(msg, args...)
}

// Info logs an info message
func (l *testLogger) Info(msg string, args ...any) {
	l.Logger.Info(msg, args...)
}

// Warn logs a warning message
func (l *testLogger) Warn(msg string, args ...any) {
	l.Logger.Warn(msg, args...)
}

// Error logs an error message
func (l *testLogger) Error(msg string, args ...any) {
	l.Logger.Error(msg, args...)
}
