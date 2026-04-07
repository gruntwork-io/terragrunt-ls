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
		name          string
		setupFunc     func(t *testing.T, tmpDir string) (source, currentDir string)
		expectedFile  string // relative to tmpDir
		shouldResolve bool
	}{
		{
			name: "directory with main.tf",
			setupFunc: func(t *testing.T, tmpDir string) (string, string) {
				t.Helper()

				moduleDir := filepath.Join(tmpDir, "modules", "database")
				require.NoError(t, os.MkdirAll(moduleDir, 0755))

				// Create main.tf in the module directory
				mainTF := filepath.Join(moduleDir, "main.tf")
				require.NoError(t, os.WriteFile(mainTF, []byte("# main.tf"), 0644))

				return "./modules/database", tmpDir
			},
			expectedFile:  "modules/database/main.tf",
			shouldResolve: true,
		},
		{
			name: "directory with other .tf files (no main.tf)",
			setupFunc: func(t *testing.T, tmpDir string) (string, string) {
				t.Helper()

				moduleDir := filepath.Join(tmpDir, "modules", "vpc")
				require.NoError(t, os.MkdirAll(moduleDir, 0755))

				// Create some .tf files (no main.tf)
				require.NoError(t, os.WriteFile(filepath.Join(moduleDir, "variables.tf"), []byte("# variables.tf"), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(moduleDir, "outputs.tf"), []byte("# outputs.tf"), 0644))

				return "./modules/vpc", tmpDir
			},
			expectedFile:  "modules/vpc", // Will be validated differently since file order isn't guaranteed
			shouldResolve: true,
		},
		{
			name: "directory with both main.tf and other files",
			setupFunc: func(t *testing.T, tmpDir string) (string, string) {
				t.Helper()

				moduleDir := filepath.Join(tmpDir, "modules", "priority-test")
				require.NoError(t, os.MkdirAll(moduleDir, 0755))

				// Create both files - main.tf should have priority
				require.NoError(t, os.WriteFile(filepath.Join(moduleDir, "main.tf"), []byte("# main.tf"), 0644))
				require.NoError(t, os.WriteFile(filepath.Join(moduleDir, "variables.tf"), []byte("# variables.tf"), 0644))

				return "./modules/priority-test", tmpDir
			},
			expectedFile:  "modules/priority-test/main.tf",
			shouldResolve: true,
		},
		{
			name: "directory with no terraform files",
			setupFunc: func(t *testing.T, tmpDir string) (string, string) {
				t.Helper()

				moduleDir := filepath.Join(tmpDir, "modules", "empty")
				require.NoError(t, os.MkdirAll(moduleDir, 0755))

				// Create a non-terraform file
				require.NoError(t, os.WriteFile(filepath.Join(moduleDir, "README.md"), []byte("# README"), 0644))

				return "./modules/empty", tmpDir
			},
			expectedFile:  "modules/empty", // Should return the directory itself
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

			resolved := definition.ResolveUnitSourceLocation(source, currentDir)

			assert.NotEmpty(t, resolved, "Resolved path should not be empty")

			if tt.shouldResolve {
				// Special handling for the "other .tf files" test case
				if tt.name == "directory with other .tf files (no main.tf)" {
					// Should resolve to one of the .tf files in the directory
					moduleDir := filepath.Join(tmpDir, "modules", "vpc")
					assert.True(t, resolved == filepath.Join(moduleDir, "variables.tf") ||
						resolved == filepath.Join(moduleDir, "outputs.tf"),
						"Resolved path should be one of the .tf files")
				} else {
					expectedPath := filepath.Join(tmpDir, tt.expectedFile)
					assert.Equal(t, expectedPath, resolved, "Resolved path should match expected")
				}

				// Verify the resolved path exists
				_, err := os.Stat(resolved)
				assert.NoError(t, err, "Resolved path should exist")
			} else {
				assert.Empty(t, resolved, "Resolved path should be empty when resolution fails")
			}
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

				// Create terragrunt.stack.hcl in the stack directory
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

				// Create a non-stack file
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

			assert.NotEmpty(t, resolved, "Resolved path should not be empty")

			if tt.shouldResolve {
				expectedPath := filepath.Join(tmpDir, tt.expectedFile)
				assert.Equal(t, expectedPath, resolved, "Resolved path should match expected")

				// Verify the resolved path exists
				_, err := os.Stat(resolved)
				assert.NoError(t, err, "Resolved path should exist")
			} else {
				assert.Empty(t, resolved, "Resolved path should be empty when resolution fails")
			}
		})
	}
}
