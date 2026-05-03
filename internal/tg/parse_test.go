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

			tmpDir := t.TempDir()
			filename := filepath.Join(tmpDir, "terragrunt.stack.hcl")

			l := testutils.NewTestLogger(t)
			cfg, diags := tg.ParseStackBuffer(t.Context(), l, filename, tt.content)

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
		{
			name: "values attribute access should not show diagnostic",
			setup: func(t *testing.T, tmpDir string) string {
				t.Helper()

				return filepath.Join(tmpDir, "terragrunt.hcl")
			},
			content: `locals {
  environment = values.environment
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

// TestParseTerragruntBuffer_DoesNotPanic is a regression test for
// gruntwork-io/terragrunt-ls#134, where opening a root.hcl that uses
// read_terragrunt_config / find_in_parent_folders together with a remote_state
// block caused the upstream Terragrunt parser to panic with a nil-pointer
// dereference, crashing the language server. Whatever the underlying behavior,
// the LS must not panic on user input — it should return whatever it can and
// log the error.
func TestParseTerragruntBuffer_DoesNotPanic(t *testing.T) {
	t.Parallel()

	// This is a reduced version of the user's reproducer from issue #134.
	// The remote_state config references locals that depend on
	// read_terragrunt_config calls whose target files do not exist, so the
	// remote_state block cannot be fully resolved during LS parsing.
	const rootHCL = `
locals {
  account_vars = read_terragrunt_config(find_in_parent_folders("account.hcl"))

  terraform_state_bucket = try(
    local.account_vars.locals.terraform_state_bucket,
    "fallback-bucket",
  )
  terraform_assume_role_arn = try(
    local.account_vars.locals.terraform_assume_role_arn,
    "",
  )
}

remote_state {
  backend = "s3"
  generate = {
    path      = "backend.tf"
    if_exists = "overwrite_terragrunt"
  }
  config = merge({
    bucket  = local.terraform_state_bucket
    key     = "${path_relative_to_include()}/terraform.tfstate"
    region  = "us-east-1"
    encrypt = true
    },
    local.terraform_assume_role_arn != "" ? {
      assume_role = {
        role_arn = local.terraform_assume_role_arn
      }
  } : {})
}
`

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "root.hcl")

	l := testutils.NewTestLogger(t)

	// The call must not panic. We deliberately do not assert on the returned
	// cfg/diags shape: the goal is to prove the LS survives malformed or
	// unresolvable inputs that previously caused a panic in the upstream
	// Terragrunt parser.
	require.NotPanics(t, func() {
		tg.ParseTerragruntBuffer(context.Background(), l, filename, rootHCL)
	})
}
