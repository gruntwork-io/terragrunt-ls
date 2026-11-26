package tg_test

import (
	"path/filepath"
	"strings"
	"terragrunt-ls/internal/testutils"
	"terragrunt-ls/internal/tg"
	"terragrunt-ls/internal/tg/store"
	"testing"

	"github.com/gruntwork-io/terragrunt/pkg/venv"

	"github.com/gruntwork-io/terragrunt/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
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
			name:     "unit autoinclude file",
			filename: "/some/path/terragrunt.autoinclude.hcl",
			expected: store.FileTypeShared,
		},
		{
			name:     "stack autoinclude file",
			filename: "/some/path/terragrunt.autoinclude.stack.hcl",
			expected: store.FileTypeUnknown,
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

func TestParseTerragruntBufferReadsTheBuffer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		setup func(t *testing.T, v *venv.Venv, tmpDir string) string
		name  string
	}{
		{
			name: "file on disk holds other content",
			setup: func(t *testing.T, v *venv.Venv, tmpDir string) string {
				t.Helper()

				path, err := testutils.CreateFile(v, tmpDir, "terragrunt.hcl", `inputs = {
  source = "disk"
}
`)
				require.NoError(t, err)

				return path
			},
		},
		{
			name: "directory does not exist",
			setup: func(t *testing.T, v *venv.Venv, tmpDir string) string {
				t.Helper()

				return filepath.Join(tmpDir, "unsaved", "terragrunt.hcl")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v := venv.MemVenv()
			filename := tt.setup(t, v, testutils.WorkDir(t, v))

			cfg, diags := tg.ParseTerragruntBuffer(
				t.Context(),
				testutils.NewTestLogger(t),
				v,
				venv.Policy{},
				filename,
				`inputs = {
  source = "buffer"
}
`,
			)

			assert.Empty(t, diags)
			require.NotNil(t, cfg)
			assert.Equal(t, map[string]any{"source": "buffer"}, cfg.Inputs)
		})
	}
}

func TestParseTerragruntBufferPathVariable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		check   func(t *testing.T, cfg *config.TerragruntConfig, rootDir, unitDir string)
		name    string
		content string
	}{
		{
			name: "path.unit in locals",
			content: `locals {
  unit_dir = path.unit
}
`,
			check: func(t *testing.T, cfg *config.TerragruntConfig, _, unitDir string) {
				t.Helper()

				assert.Equal(t, unitDir, cfg.Locals["unit_dir"])
			},
		},
		{
			name: "path.download in inputs",
			content: `inputs = {
  download_dir = path.download
}
`,
			check: func(t *testing.T, cfg *config.TerragruntConfig, _, unitDir string) {
				t.Helper()

				downloadDir, ok := cfg.Inputs["download_dir"].(string)
				require.True(t, ok)
				assert.True(
					t,
					strings.HasPrefix(downloadDir, filepath.Join(unitDir, ".terragrunt-cache")+string(filepath.Separator)),
				)
			},
		},
		{
			name: "path.include and path.unit in an included file",
			content: `include "root" {
  path = find_in_parent_folders("root.hcl")
}
`,
			check: func(t *testing.T, cfg *config.TerragruntConfig, rootDir, unitDir string) {
				t.Helper()

				assert.Equal(t, filepath.Join(rootDir, "root.hcl"), cfg.Inputs["root_file"])
				assert.Equal(t, unitDir, cfg.Inputs["unit_dir"])
			},
		},
		{
			name: "get_original_terragrunt_dir in locals",
			content: `locals {
  unit_dir = get_original_terragrunt_dir()
}
`,
			check: func(t *testing.T, cfg *config.TerragruntConfig, _, unitDir string) {
				t.Helper()

				assert.Equal(t, unitDir, cfg.Locals["unit_dir"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v := venv.MemVenv()
			rootDir := testutils.WorkDir(t, v)
			unitDir := filepath.Join(rootDir, "app")

			_, err := testutils.CreateFile(v, rootDir, "root.hcl", `inputs = {
  root_file = path.include
  unit_dir  = path.unit
}
`)
			require.NoError(t, err)

			cfg, diags := tg.ParseTerragruntBuffer(
				t.Context(),
				testutils.NewTestLogger(t),
				v,
				venv.Policy{},
				filepath.Join(unitDir, "terragrunt.hcl"),
				tt.content,
			)

			assert.Empty(t, diags)
			require.NotNil(t, cfg)
			tt.check(t, cfg, rootDir, unitDir)
		})
	}
}

func TestParseTerragruntBufferPathVariableErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		message   func(filename string) string
		name      string
		content   string
		diagRange protocol.Range
		severity  protocol.DiagnosticSeverity
	}{
		{
			name: "path.stack in a unit",
			content: `locals {
  stack_dir = path.stack
}
`,
			severity: protocol.DiagnosticSeverityError,
			diagRange: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 18},
				End:   protocol.Position{Line: 1, Character: 24},
			},
			message: func(string) string {
				return `Unsupported attribute: This object does not have an attribute named "stack".`
			},
		},
		{
			name: "path.download in locals",
			content: `locals {
  unit_dir     = path.unit
  download_dir = path.download
}
`,
			severity: protocol.DiagnosticSeverityError,
			diagRange: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 2},
				End:   protocol.Position{Line: 2, Character: 30},
			},
			message: func(filename string) string {
				return config.PathDownloadInLocalsError{ConfigPath: filename, Local: "download_dir"}.Error()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v := venv.MemVenv()
			filename := filepath.Join(testutils.WorkDir(t, v), "terragrunt.hcl")

			_, diags := tg.ParseTerragruntBuffer(
				t.Context(),
				testutils.NewTestLogger(t),
				v,
				venv.Policy{},
				filename,
				tt.content,
			)

			require.Len(t, diags, 1)
			assert.Equal(t, tt.severity, diags[0].Severity)
			assert.Equal(t, tt.diagRange, diags[0].Range)
			assert.Equal(t, protocol.String(tt.message(filename)), diags[0].Message)
		})
	}
}

func TestParseTerragruntBufferPathDownloadInIncludedLocals(t *testing.T) {
	t.Parallel()

	v := venv.MemVenv()
	rootDir := testutils.WorkDir(t, v)

	_, err := testutils.CreateFile(v, rootDir, "root.hcl", "locals {\n  download_dir = path.download\n}\n")
	require.NoError(t, err)

	_, diags := tg.ParseTerragruntBuffer(
		t.Context(),
		testutils.NewTestLogger(t),
		v,
		venv.Policy{},
		filepath.Join(rootDir, "app", "terragrunt.hcl"),
		unitIncludingRoot+"\nlocals {\n  download_dir = path.unit\n}\n",
	)

	require.Len(t, diags, 1)
	assert.Equal(t, protocol.DiagnosticSeverityError, diags[0].Severity)
	assert.Equal(t, protocol.NewOptional("terragrunt-ls"), diags[0].Source)
	assert.Equal(t, protocol.Range{}, diags[0].Range)
}

func TestParseTerragruntBufferPathVariableInIncludedFile(t *testing.T) {
	t.Parallel()

	v := venv.MemVenv()
	rootDir := testutils.WorkDir(t, v)

	_, err := testutils.CreateFile(v, rootDir, "root.hcl", "inputs = { stack_dir = path.stack }\n")
	require.NoError(t, err)

	_, diags := tg.ParseTerragruntBuffer(
		t.Context(),
		testutils.NewTestLogger(t),
		v,
		venv.Policy{},
		filepath.Join(rootDir, "app", "terragrunt.hcl"),
		unitIncludingRoot,
	)

	assert.Empty(t, diags)
}

func TestParseTerragruntBufferPathVariableSharedFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		content  string
		messages []string
	}{
		{
			name: "fields of an including or reading file",
			content: `locals {
  include_path = path.include
}

inputs = {
  read_path = path.read
  stack_dir = path.stack
}
`,
			messages: []string{},
		},
		{
			name: "try with the fields of an including or reading file",
			content: `locals {
  common_dir = dirname(try(path.include, path.read))
}
`,
			messages: []string{},
		},
		{
			name: "try with a function call on each field",
			content: `locals {
  common_dir = try(dirname(path.include), dirname(path.read))
}
`,
			messages: []string{},
		},
		{
			name: "index form of a field",
			content: `inputs = {
  dir = path["include"]
}
`,
			messages: []string{},
		},
		{
			name: "field in a typed attribute",
			content: `terraform {
  source = "${dirname(path.include)}/modules/x"
}
`,
			messages: []string{},
		},
		{
			name: "try in a typed attribute",
			content: `terraform {
  source = "${dirname(try(path.include, path.read))}/modules/x"
}
`,
			messages: []string{},
		},
		{
			name: "try in a local that a typed attribute reads",
			content: `locals {
  common_dir = dirname(try(path.include, path.read))
}

terraform {
  source = "${local.common_dir}/modules/x"
}
`,
			messages: []string{},
		},
		{
			name: "field in a chain of locals that a typed attribute reads",
			content: `locals {
  root_file = path.include
  root_dir  = dirname(local.root_file)
}

download_dir = "${local.root_dir}/cache"
`,
			messages: []string{},
		},
		{
			name: "unknown field in a local that a typed attribute reads",
			content: `locals {
  root_file = path.include
  typo_dir  = path.unti
}

download_dir = "${local.typo_dir}/cache"
`,
			messages: []string{
				`Unsupported attribute: This object does not have an attribute named "unti".`,
				"Unsuitable value type: Unsuitable value: value must be known",
			},
		},
		{
			name: "unknown field",
			content: `inputs = {
  dir = path.unti
}
`,
			messages: []string{`Unsupported attribute: This object does not have an attribute named "unti".`},
		},
		{
			name: "unknown field in a typed attribute",
			content: `terraform {
  source = "${path.unti}/modules/x"
}
`,
			messages: []string{
				`Unsupported attribute: This object does not have an attribute named "unti".`,
				"Unsuitable value type: Unsuitable value: value must be known",
			},
		},
		{
			name: "attribute of a field",
			content: `inputs = {
  dir = path.include.missing
}
`,
			messages: []string{`Unsupported attribute: This object does not have an attribute named "include".`},
		},
		{
			name: "field of a for variable",
			content: `inputs = {
  dirs = [for path in [{}] : path.include]
}
`,
			messages: []string{`Unsupported attribute: This object does not have an attribute named "include".`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v := venv.MemVenv()

			_, diags := tg.ParseTerragruntBuffer(
				t.Context(),
				testutils.NewTestLogger(t),
				v,
				venv.Policy{},
				filepath.Join(testutils.WorkDir(t, v), "root.hcl"),
				tt.content,
			)

			require.Len(t, diags, len(tt.messages))

			for i, message := range tt.messages {
				assert.Equal(t, protocol.String(message), diags[i].Message)
			}
		})
	}
}

func TestParseTerragruntBufferPathVariableSharedFileInvalidTry(t *testing.T) {
	t.Parallel()

	v := venv.MemVenv()

	_, diags := tg.ParseTerragruntBuffer(
		t.Context(),
		testutils.NewTestLogger(t),
		v,
		venv.Policy{},
		filepath.Join(testutils.WorkDir(t, v), "root.hcl"),
		`inputs = {
  dir = try(path.include.missing, path.read.missing)
}
`,
	)

	require.Len(t, diags, 1)
	assert.Equal(t, protocol.DiagnosticSeverityError, diags[0].Severity)
	assert.Equal(
		t,
		protocol.Range{
			Start: protocol.Position{Line: 1, Character: 8},
			End:   protocol.Position{Line: 1, Character: 12},
		},
		diags[0].Range,
	)
}

func TestParseStackBufferPathVariable(t *testing.T) {
	t.Parallel()

	v := venv.MemVenv()
	stackDir := testutils.WorkDir(t, v)

	cfg, diags := tg.ParseStackBuffer(
		t.Context(),
		testutils.NewTestLogger(t),
		v,
		venv.Policy{},
		filepath.Join(stackDir, "terragrunt.stack.hcl"),
		`locals {
  stack_dir = path.stack
}

unit "vpc" {
  source = "./units/vpc"
  path   = "vpc"
}
`,
	)

	assert.Empty(t, diags)
	require.NotNil(t, cfg)
	assert.Equal(t, stackDir, cfg.Locals["stack_dir"])
}

func TestParseStackBufferPathVariableErrors(t *testing.T) {
	t.Parallel()

	v := venv.MemVenv()

	_, diags := tg.ParseStackBuffer(
		t.Context(),
		testutils.NewTestLogger(t),
		v,
		venv.Policy{},
		filepath.Join(testutils.WorkDir(t, v), "terragrunt.stack.hcl"),
		`locals {
  include_path = path.include
}
`,
	)

	require.Len(t, diags, 1)
	assert.Equal(
		t,
		protocol.String(`Unsupported attribute: This object does not have an attribute named "include".`),
		diags[0].Message,
	)
}
