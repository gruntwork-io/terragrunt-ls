// Package testutils provides utilities for testing.
package testutils

import (
	"log/slog"
	"os"
	"path/filepath"
	"terragrunt-ls/internal/logger"
	"testing"
)

func NewTestLogger(t *testing.T) *logger.Logger {
	t.Helper()

	// Create a test logger that writes to the test log
	testWriter := testWriter{t}
	handler := slog.NewJSONHandler(testWriter, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	slogger := slog.New(handler)

	// Create a new logger with the test writer
	return &logger.Logger{
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
