package tg_test

import (
	"os"
	"path/filepath"
	"terragrunt-ls/internal/testutils"
	"terragrunt-ls/internal/tg"
	"testing"

	"github.com/gruntwork-io/terragrunt/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func TestParseTerragruntBuffer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		setup    func(t *testing.T, tmpDir string) string
		wantCfg  *config.TerragruntConfig
		name     string
		content  string
		wantDiag bool
	}{
		{
			name: "basic terragrunt config",
			setup: func(t *testing.T, tmpDir string) string {
				t.Helper()

				path := filepath.Join(tmpDir, "terragrunt.hcl")
				return path
			},
			content: `
terraform {
  source = "./modules/example"
}
`,
			wantDiag: false,
		},
		{
			name: "terragrunt stack file",
			setup: func(t *testing.T, tmpDir string) string {
				t.Helper()

				path := filepath.Join(tmpDir, "terragrunt.stack.hcl")
				return path
			},
			content: `
unit "database" {
  source = "git::git@github.com:acme/infrastructure-catalog.git//units/mysql"
  path   = "database"
}

unit "app" {
  source = "git::git@github.com:acme/infrastructure-catalog.git//units/app"
  path   = "app"
}
`,
			wantDiag: false,
		},
		{
			name: "terragrunt values file",
			setup: func(t *testing.T, tmpDir string) string {
				t.Helper()

				path := filepath.Join(tmpDir, "terragrunt.values.hcl")
				return path
			},
			content: `
values {
  instance_type = "t3.micro"
  environment   = "dev"
}

dependency "vpc" {
  config_path = "../vpc"

  mock_outputs = {
    vpc_id = "vpc-1234567890"
  }
}
`,
			wantDiag: false,
		},
		{
			name: "dependency with missing outputs should not show diagnostic",
			setup: func(t *testing.T, tmpDir string) string {
				t.Helper()

				// Create base module directory and its terragrunt.hcl
				baseDir := filepath.Join(tmpDir, "base")
				require.NoError(t, os.MkdirAll(baseDir, 0755))

				baseTg := filepath.Join(baseDir, "terragrunt.hcl")
				require.NoError(t, os.WriteFile(baseTg, []byte(`
terraform {
  source = "./module"
}
`), 0644))

				// Create unit directory and return path to its terragrunt.hcl
				unitDir := filepath.Join(tmpDir, "unit")
				require.NoError(t, os.MkdirAll(unitDir, 0755))
				return filepath.Join(unitDir, "terragrunt.hcl")
			},
			content: `
dependency "base" {
  config_path = "../base"

  mock_outputs = {
    vpc_id = "vpc-1234567890"
  }
}

terraform {
  source = "./modules/example"
}

inputs = {
  vpc_id = dependency.base.outputs.vpc_id
}
`,
			wantDiag: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create temporary directory for test
			tmpDir := t.TempDir()

			// Run setup and get file to parse
			filename := tt.setup(t, tmpDir)

			l := testutils.NewTestLogger(t)

			// Call the appropriate parser based on file type
			fileType := tg.GetTerragruntFileType(filename)
			var diags []protocol.Diagnostic

			switch fileType {
			case tg.TerragruntFileTypeUnknown:
				t.Fatalf("Unknown file type for: %s", filename)

			case tg.TerragruntFileTypeStack:
				stackCfg, stackDiags := tg.ParseTerragruntStackBuffer(l, filename, tt.content)
				diags = stackDiags

				if tt.wantDiag {
					assert.NotEmpty(t, diags, "expected diagnostics but got none")
				} else {
					assert.Empty(t, diags, "expected no diagnostics but got: %v", diags)
				}

				// For stack files, we should get a valid StackConfig
				assert.NotNil(t, stackCfg, "expected stack config to not be nil for stack files")

			case tg.TerragruntFileTypeValues:
				cfg, valuesDiags := tg.ParseTerragruntValuesBuffer(l, filename, tt.content)
				diags = valuesDiags

				if tt.wantDiag {
					assert.NotEmpty(t, diags, "expected diagnostics but got none")
				} else {
					assert.Empty(t, diags, "expected no diagnostics but got: %v", diags)
				}

				// For values files, we should get a valid HCL file
				assert.NotNil(t, cfg, "expected valid HCL config for values files")

			case tg.TerragruntFileTypeConfig:
				cfg, configDiags := tg.ParseTerragruntConfigBuffer(l, filename, tt.content)
				diags = configDiags

				if tt.wantDiag {
					assert.NotEmpty(t, diags, "expected diagnostics but got none")
				} else {
					assert.Empty(t, diags, "expected no diagnostics but got: %v", diags)
				}

				// For regular terragrunt files, we should get a valid TerragruntConfig
				assert.NotNil(t, cfg, "expected config to not be nil for regular terragrunt files")

			default:
				t.Fatalf("Unknown file type for: %s", filename)
			}
		})
	}
}

func TestGetTerragruntFileType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		filename string
		name     string
		expected tg.TerragruntFileType
	}{
		{
			name:     "terragrunt.hcl file",
			filename: "/path/to/terragrunt.hcl",
			expected: tg.TerragruntFileTypeConfig,
		},
		{
			name:     "terragrunt.stack.hcl file",
			filename: "/path/to/terragrunt.stack.hcl",
			expected: tg.TerragruntFileTypeStack,
		},
		{
			name:     "terragrunt.values.hcl file",
			filename: "/path/to/terragrunt.values.hcl",
			expected: tg.TerragruntFileTypeValues,
		},
		{
			name:     "other .hcl file",
			filename: "/path/to/variables.hcl",
			expected: tg.TerragruntFileTypeConfig,
		},
		{
			name:     "non-hcl file",
			filename: "/path/to/main.tf",
			expected: tg.TerragruntFileTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tg.GetTerragruntFileType(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}
