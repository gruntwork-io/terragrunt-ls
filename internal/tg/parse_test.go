package tg_test

import (
	"path/filepath"
	"terragrunt-ls/internal/testutils"
	"terragrunt-ls/internal/tg"
	"terragrunt-ls/internal/tg/store"
	"testing"

	"github.com/gruntwork-io/terragrunt/pkg/venv"

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
			expected: store.FileTypeUnit,
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
			expected: store.FileTypeUnknown,
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
			wantDiag: false,
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

			v := venv.MemVenv()

			tmpDir := testutils.WorkDir(t, v)
			filename := filepath.Join(tmpDir, "terragrunt.stack.hcl")

			l := testutils.NewTestLogger(t)
			cfg, diags := tg.ParseStackBuffer(t.Context(), l, v, venv.Policy{}, filename, tt.content)

			if tt.wantDiag {
				assert.NotEmpty(t, diags, "expected diagnostics but got none")
			} else {
				assert.Empty(t, diags, "expected no diagnostics but got: %v", diags)
			}

			if tt.wantCfg {
				assert.NotNil(t, cfg, "expected config to not be nil")
			} else {
				assert.Nil(t, cfg, "expected config to be nil")
			}
		})
	}
}

func TestParseTerragruntBuffer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		setup    func(t *testing.T, v *venv.Venv, tmpDir string) string
		wantCfg  *config.TerragruntConfig
		name     string
		content  string
		wantDiag bool
	}{
		{
			name: "basic terragrunt config",
			setup: func(t *testing.T, v *venv.Venv, tmpDir string) string {
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
			setup: func(t *testing.T, v *venv.Venv, tmpDir string) string {
				t.Helper()

				// Create base module directory and its terragrunt.hcl
				baseDir := filepath.Join(tmpDir, "base")
				require.NoError(t, v.FS.MkdirAll(baseDir, 0755))

				_, err := testutils.CreateFile(v, baseDir, "terragrunt.hcl", `
terraform {
  source = "./module"
}
`)
				require.NoError(t, err)

				// Create unit directory and return path to its terragrunt.hcl
				unitDir := filepath.Join(tmpDir, "unit")
				require.NoError(t, v.FS.MkdirAll(unitDir, 0755))

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
		{
			name: "values attribute access should not show diagnostic",
			setup: func(t *testing.T, v *venv.Venv, tmpDir string) string {
				t.Helper()

				return filepath.Join(tmpDir, "terragrunt.hcl")
			},
			content: `locals {
  environment = values.environment
}
`,
			wantDiag: false,
		},
		{
			name: "bare include block",
			setup: func(t *testing.T, v *venv.Venv, tmpDir string) string {
				t.Helper()

				_, err := testutils.CreateFile(v, tmpDir, "root.hcl", `
locals {
  region = "us-east-1"
}
`)
				require.NoError(t, err)

				unitDir := filepath.Join(tmpDir, "unit")
				require.NoError(t, v.FS.MkdirAll(unitDir, 0755))

				return filepath.Join(unitDir, "terragrunt.hcl")
			},
			content: `
include {
  path = find_in_parent_folders("root.hcl")
}

inputs = {
  name = "example"
}
`,
			wantDiag: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v := venv.MemVenv()
			tmpDir := testutils.WorkDir(t, v)

			// Run setup and get file to parse
			filename := tt.setup(t, v, tmpDir)

			l := testutils.NewTestLogger(t)
			cfg, diags := tg.ParseTerragruntBuffer(
				t.Context(),
				l,
				v,
				venv.Policy{},
				filename,
				tt.content,
			)

			if tt.wantDiag {
				assert.NotEmpty(t, diags, "expected diagnostics but got none")
			} else {
				assert.Empty(t, diags, "expected no diagnostics but got: %v", diags)
			}

			assert.NotNil(t, cfg, "expected config to not be nil")
		})
	}
}
