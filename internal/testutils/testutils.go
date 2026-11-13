// Package testutils provides utilities for testing.
package testutils

import (
	"errors"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"terragrunt-ls/internal/docs"
	"terragrunt-ls/internal/logger"
	"testing"
	"testing/fstest"

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

// DocsUnitsIndex is the index page of unit configuration items that [NewDocs]
// starts from.
const DocsUnitsIndex = "# Units\n\n" +
	"## Blocks\n\n" +
	"| Name | Description |\n" +
	"| --- | --- |\n" +
	"| [`dependency`](/reference/hcl/blocks/dependency/) | Reads the outputs of another unit. |\n" +
	"| [`dependencies`](/reference/hcl/blocks/dependencies/) | Lists the units to run before this one. |\n" +
	"| [`generate`](/reference/hcl/blocks/generate/) | Writes a file into the working directory. |\n" +
	"| [`include`](/reference/hcl/blocks/include/) | Merges another configuration file into this one. |\n" +
	"| [`locals`](/reference/hcl/blocks/locals/) | Defines values to reuse in the file. |\n\n" +
	"## Attributes\n\n" +
	"| Name | Description |\n" +
	"| --- | --- |\n" +
	"| [`inputs`](/reference/hcl/attributes/inputs/) | Sets the input variables of the module. |\n\n" +
	"## Functions\n\n" +
	"| Name | Description |\n" +
	"| --- | --- |\n" +
	"| [`get_env`](/reference/hcl/functions/get_env/) | Returns the value of an environment variable. |\n"

// DocsStacksIndex is the index page of stack configuration items that
// [NewDocs] starts from.
const DocsStacksIndex = "# Stacks\n\n" +
	"## Blocks\n\n" +
	"| Name | Description |\n" +
	"| --- | --- |\n" +
	"| [`locals`](/reference/hcl/blocks/locals/) | Defines values to reuse in the file. |\n" +
	"| [`stack`](/reference/hcl/blocks/stack/) | Defines a nested stack. |\n" +
	"| [`unit`](/reference/hcl/blocks/unit/) | Defines a unit. |\n\n" +
	"## Functions\n\n" +
	"| Name | Description |\n" +
	"| --- | --- |\n" +
	"| [`get_env`](/reference/hcl/functions/get_env/) | Returns the value of an environment variable. |\n"

// DocsSnapshot returns a snapshot holding [DocsUnitsIndex] and
// [DocsStacksIndex].
func DocsSnapshot() fstest.MapFS {
	return fstest.MapFS{
		"units.md":  {Data: []byte(DocsUnitsIndex)},
		"stacks.md": {Data: []byte(DocsStacksIndex)},
	}
}

// DocsBlurb returns what [docs.Docs.Blurb] returns for an item with
// description whose page is at `/reference/hcl/<page>/` on
// [docs.DefaultBaseURL], e.g. `blocks/locals`.
func DocsBlurb(description, page string) string {
	return description + "\n\n[Terragrunt docs](" + docs.DefaultBaseURL + "/reference/hcl/" + page + "/)"
}

// NewDocs returns a reference that starts from [DocsSnapshot], links to
// [docs.DefaultBaseURL], and fetches nothing.
func NewDocs(t *testing.T) *docs.Docs {
	t.Helper()

	base, err := url.Parse(docs.DefaultBaseURL)
	require.NoError(t, err)

	d, err := docs.New(&docs.Options{Snapshot: DocsSnapshot(), BaseURL: base, Logger: NewTestLogger(t)})
	require.NoError(t, err)

	return d
}
