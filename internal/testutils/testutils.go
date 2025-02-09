// Package testutils provides utilities for testing.
package testutils

import (
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func NewTestLogger(t *testing.T) *zap.SugaredLogger {
	t.Helper()

	l := zaptest.NewLogger(t)

	return zap.New(l.Core(), zap.AddCaller()).Sugar()
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
