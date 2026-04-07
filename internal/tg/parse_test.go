package tg_test

import (
	"context"
	"os"
	"path/filepath"
	"terragrunt-ls/internal/testutils"
	"terragrunt-ls/internal/tg"
	"terragrunt-ls/internal/tg/store"
	"testing"

	"github.com/gruntwork-io/terragrunt/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectFileType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filename string
		expected store.FileType
	}{
		{
			name:     "regular terragrunt file",
			filename: "/some/path/terragrunt.hcl",
			expected: store.FileTypeTerragrunt,
		},
		{
			name:     "stack file",
			filename: "/some/path/terragrunt.stack.hcl",
			expected: store.FileTypeStack,
		},
		{
			name:     "values file",
			filename: "/some/path/terragrunt.values.hcl",
			expected: store.FileTypeValues,
		},
		{
			name:     "arbitrary hcl file",
			filename: "/some/path/other.hcl",
			expected: store.FileTypeTerragrunt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tg.DetectFileType(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseStackBuffer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		content  string
		wantCfg  bool
		wantDiag bool
	}{
		{
			name: "valid stack config with unit",
			content: `unit "vpc" {
	source = "./units/vpc"
	path   = "vpc"
}`,
			wantCfg:  true,
			wantDiag: false,
		},
		{
			name: "valid stack config with stack and unit",
			content: `unit "vpc" {
	source = "./units/vpc"
	path   = "vpc"
}

stack "service" {
	source = "./stacks/service"
	path   = "service"
}`,
			wantCfg:  true,
			wantDiag: false,
		},
		{
			name:     "empty stack config",
			content:  "",
			wantCfg:  false,
			wantDiag: true,
		},
		{
			name: "unit missing source",
			content: `unit "vpc" {
	path = "vpc"
}`,
			wantCfg:  false,
			wantDiag: true,
		},
		{
			name: "unit missing path",
			content: `unit "vpc" {
	source = "./units/vpc"
}`,
			wantCfg:  false,
			wantDiag: true,
		},
		{
			name:     "unit missing both source and path",
			content:  `unit "vpc" {}`,
			wantCfg:  false,
			wantDiag: true,
		},
		{
			name: "stack missing source",
			content: `stack "service" {
	path = "service"
}`,
			wantCfg:  false,
			wantDiag: true,
		},
		{
			name: "stack missing path",
			content: `stack "service" {
	source = "./stacks/service"
}`,
			wantCfg:  false,
			wantDiag: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			filename := filepath.Join(tmpDir, "terragrunt.stack.hcl")

			l := testutils.NewTestLogger(t)
			cfg, diags := tg.ParseStackBuffer(l, filename, tt.content)

			if tt.wantDiag {
				assert.NotEmpty(t, diags, "expected diagnostics but got none")
			} else {
				assert.Empty(t, diags, "expected no diagnostics but got: %v", diags)
			}

			if tt.wantCfg {
				assert.NotNil(t, cfg, "expected config to not be nil")
			}
		})
	}
}

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
			cfg, diags := tg.ParseTerragruntBuffer(context.Background(), l, filename, tt.content)

			if tt.wantDiag {
				assert.NotEmpty(t, diags, "expected diagnostics but got none")
			} else {
				assert.Empty(t, diags, "expected no diagnostics but got: %v", diags)
			}

			assert.NotNil(t, cfg, "expected config to not be nil")
		})
	}
}
