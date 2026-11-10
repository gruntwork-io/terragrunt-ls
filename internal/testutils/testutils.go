// Package testutils provides utilities for testing.
package testutils

import (
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"terragrunt-ls/internal/logger"
	"testing"

	"github.com/gruntwork-io/terragrunt/pkg/venv"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/uri"
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

// CreateFile writes content to the file name in dir on the filesystem of v. It
// creates dir when dir does not exist.
func CreateFile(v *venv.Venv, dir, name, content string) (string, error) {
	const (
		ownerRWXGlobalRX = 0755
		ownerRWGlobalR   = 0644
	)

	if err := v.FS.MkdirAll(dir, ownerRWXGlobalRX); err != nil {
		return "", err
	}

	path := filepath.Join(dir, name)

	f, err := v.FS.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, ownerRWGlobalR)
	if err != nil {
		return "", err
	}

	if _, err := f.WriteString(content); err != nil {
		return "", errors.Join(err, f.Close())
	}

	return path, f.Close()
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

// WorkDir returns the working directory of v, which is an absolute path on
// every platform.
//
// The path is spelled as a document URI spells it. On Windows that means a
// lowercase drive letter, and the in-memory filesystem is case-sensitive, so a
// file seeded under any other spelling is not found through a URI.
func WorkDir(t *testing.T, v *venv.Venv) string {
	t.Helper()

	dir, err := v.Platform.Getwd()
	require.NoError(t, err)

	return uri.File(dir).FsPath()
}
