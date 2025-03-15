// Package testutils provides utilities for testing.
package testutils

import (
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"terragrunt-ls/internal/logger"
	"testing"

	"github.com/stretchr/testify/require"
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

// CloneGitRepo clones a git repo into a temporary directory,
// and returns the path to the cloned repo.
func CloneGitRepo(t *testing.T, repoURL, branch string) string {
	t.Helper()

	tempdir := t.TempDir()

	cmd := exec.Command("git", "clone", "--branch", branch, "--depth", "1", repoURL, tempdir)

	err := cmd.Run()
	require.NoError(t, err)

	return tempdir
}

// HCLFilesInDir returns a list of all HCL files in a directory.
func HCLFilesInDir(t *testing.T, dir string) []string {
	t.Helper()

	files := []string{}

	walkFn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, ".hcl") {
			files = append(files, path)
		}

		return nil
	}

	err := filepath.WalkDir(dir, walkFn)
	require.NoError(t, err)

	return files
}

// ReadFile reads a file and returns its contents.
func ReadFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	require.NoError(t, err)

	return string(content)
}
