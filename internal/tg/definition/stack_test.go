package definition_test

import (
	"os"
	"path/filepath"
	"terragrunt-ls/internal/tg/definition"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveUnitSourceLocation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		source       string
		expectedFile string // relative to tmpDir
		files        []string
	}{
		{
			name:         "directory with terragrunt.hcl",
			files:        []string{"modules/vpc/terragrunt.hcl"},
			source:       "./modules/vpc",
			expectedFile: "modules/vpc/terragrunt.hcl",
		},
		{
			name:         "terragrunt.hcl is preferred over other files",
			files:        []string{"modules/vpc/terragrunt.hcl", "modules/vpc/main.tf"},
			source:       "./modules/vpc",
			expectedFile: "modules/vpc/terragrunt.hcl",
		},
		{
			name:         "directory without terragrunt.hcl resolves to the directory",
			files:        []string{"modules/vpc/main.tf"},
			source:       "./modules/vpc",
			expectedFile: "modules/vpc",
		},
		{
			name:         "non-existent directory",
			source:       "./non-existent",
			expectedFile: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()

			for _, f := range tt.files {
				path := filepath.Join(tmpDir, f)
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
				require.NoError(t, os.WriteFile(path, []byte(""), 0644))
			}

			resolved := definition.ResolveUnitSourceLocation(tt.source, tmpDir)

			if tt.expectedFile == "" {
				assert.Empty(t, resolved)
				return
			}

			assert.Equal(t, filepath.Join(tmpDir, tt.expectedFile), resolved)
		})
	}
}

func TestResolveStackSourceLocation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setupFunc     func(t *testing.T, tmpDir string) (source, currentDir string)
		expectedFile  string // relative to tmpDir
		shouldResolve bool
	}{
		{
			name: "directory with terragrunt.stack.hcl",
			setupFunc: func(t *testing.T, tmpDir string) (string, string) {
				t.Helper()

				stackDir := filepath.Join(tmpDir, "stacks", "webapp")
				require.NoError(t, os.MkdirAll(stackDir, 0755))

				stackHCL := filepath.Join(stackDir, "terragrunt.stack.hcl")
				require.NoError(t, os.WriteFile(stackHCL, []byte("# terragrunt.stack.hcl"), 0644))

				return "./stacks/webapp", tmpDir
			},
			expectedFile:  "stacks/webapp/terragrunt.stack.hcl",
			shouldResolve: true,
		},
		{
			name: "directory with no stack file",
			setupFunc: func(t *testing.T, tmpDir string) (string, string) {
				t.Helper()

				stackDir := filepath.Join(tmpDir, "stacks", "empty")
				require.NoError(t, os.MkdirAll(stackDir, 0755))

				require.NoError(t, os.WriteFile(filepath.Join(stackDir, "README.md"), []byte("# README"), 0644))

				return "./stacks/empty", tmpDir
			},
			expectedFile:  "stacks/empty", // Should return the directory itself
			shouldResolve: true,
		},
		{
			name: "non-existent directory",
			setupFunc: func(t *testing.T, tmpDir string) (string, string) {
				t.Helper()

				return "./non-existent", tmpDir
			},
			expectedFile:  "",
			shouldResolve: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			source, currentDir := tt.setupFunc(t, tmpDir)

			resolved := definition.ResolveStackSourceLocation(source, currentDir)

			if tt.shouldResolve {
				expectedPath := filepath.Join(tmpDir, tt.expectedFile)
				assert.Equal(t, expectedPath, resolved, "Resolved path should match expected")

				_, err := os.Stat(resolved)
				assert.NoError(t, err, "Resolved path should exist")
			} else {
				assert.Empty(t, resolved, "Resolved path should be empty when resolution fails")
			}
		})
	}
}
