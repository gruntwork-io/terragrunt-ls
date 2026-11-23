package definition_test

import (
	"os"
	"path/filepath"
	"terragrunt-ls/internal/testutils"
	"terragrunt-ls/internal/tg"
	"testing"

	"github.com/gruntwork-io/terragrunt/pkg/venv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

const basicStack = `unit "database" {
  source = "./database"
  path   = "db"
}

stack "nested" {
  source = "./nested-stack"
  path   = "nested"
}`

func TestStackDefinition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		target  string
		files   []string
		dirs    []string
		line    uint32
		char    uint32
	}{
		{
			name:    "unit source",
			content: basicStack,
			files:   []string{"database/terragrunt.hcl"},
			line:    1,
			char:    15,
			target:  "database/terragrunt.hcl",
		},
		{
			name:    "unit source without terragrunt.hcl",
			content: basicStack,
			files:   []string{"database/main.tf"},
			line:    1,
			char:    15,
		},
		{
			name:    "unit source whose terragrunt.hcl is a directory",
			content: basicStack,
			dirs:    []string{"database/terragrunt.hcl"},
			line:    1,
			char:    15,
		},
		{
			name:    "unit source that does not exist",
			content: basicStack,
			line:    1,
			char:    15,
		},
		{
			name:    "stack source",
			content: basicStack,
			files:   []string{"nested-stack/terragrunt.stack.hcl"},
			line:    6,
			char:    15,
			target:  "nested-stack/terragrunt.stack.hcl",
		},
		{
			name:    "stack source with only terragrunt.hcl",
			content: basicStack,
			files:   []string{"nested-stack/terragrunt.hcl"},
			line:    6,
			char:    15,
		},
		{
			name:    "unit path",
			content: basicStack,
			files:   []string{".terragrunt-stack/db/terragrunt.hcl"},
			line:    2,
			char:    12,
			target:  ".terragrunt-stack/db/terragrunt.hcl",
		},
		{
			name:    "unit path whose generated config is a directory",
			content: basicStack,
			dirs:    []string{".terragrunt-stack/db/terragrunt.hcl"},
			line:    2,
			char:    12,
		},
		{
			name:    "unit path before generation",
			content: basicStack,
			line:    2,
			char:    12,
		},
		{
			name:    "stack path",
			content: basicStack,
			files:   []string{".terragrunt-stack/nested/terragrunt.stack.hcl"},
			line:    7,
			char:    12,
			target:  ".terragrunt-stack/nested/terragrunt.stack.hcl",
		},
		{
			name: "computed unit path",
			content: `locals {
  name = "db"
}

unit "database" {
  source = "./database"
  path   = local.name
}`,
			files:  []string{".terragrunt-stack/db/terragrunt.hcl"},
			line:   6,
			char:   14,
			target: ".terragrunt-stack/db/terragrunt.hcl",
		},
		{
			name: "no_dot_terragrunt_stack unit path",
			content: `unit "database" {
  source                  = "./database"
  path                    = "db"
  no_dot_terragrunt_stack = true
}`,
			files:  []string{"db/terragrunt.hcl"},
			line:   2,
			char:   30,
			target: "db/terragrunt.hcl",
		},
		{
			name: "expanded unit path skips instances that were not generated",
			content: `unit "app" {
  expansion {
    for_each = toset(["a", "b"])
  }

  source = "./app"
  path   = "app-${each.key}"
}`,
			files:  []string{".terragrunt-stack/app-b/terragrunt.hcl"},
			line:   6,
			char:   12,
			target: ".terragrunt-stack/app-b/terragrunt.hcl",
		},
		{
			name: "disabled unit path",
			content: `unit "database" {
  source  = "./database"
  path    = "db"
  enabled = false
}`,
			files: []string{".terragrunt-stack/db/terragrunt.hcl"},
			line:  2,
			char:  12,
		},
		{
			name: "unit injected by autoinclude",
			content: `unit "app" {
  source = "./app"
  path   = "app"
}

stack "nested" {
  source = "./nested-stack"
  path   = "nested"

  autoinclude {
    unit "app" {
      source = "./app"
      path   = "app"
    }
  }
}`,
			files: []string{"app/terragrunt.hcl", ".terragrunt-stack/app/terragrunt.hcl"},
			line:  12,
			char:  16,
		},
		{
			name:    "unit keyword with generated config",
			content: basicStack,
			files:   []string{".terragrunt-stack/db/terragrunt.hcl"},
			line:    0,
			char:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()

			for _, f := range tt.files {
				path := filepath.Join(tmpDir, f)
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
				require.NoError(t, os.WriteFile(path, nil, 0o644))
			}

			for _, d := range tt.dirs {
				require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, d), 0o755))
			}

			docURI := uri.File(filepath.Join(tmpDir, "terragrunt.stack.hcl"))
			l := testutils.NewTestLogger(t)
			v := venv.OSVenv()

			state := tg.NewState(testutils.NewDocs(t))
			state.OpenDocument(t.Context(), l, v, venv.Policy{}, docURI, tt.content)

			position := protocol.Position{Line: tt.line, Character: tt.char}
			got := state.Definition(l, v, 1, docURI, position).Result

			// With no target, Definition answers with the cursor position.
			if tt.target == "" {
				assert.Equal(t, protocol.Location{
					URI:   docURI,
					Range: protocol.Range{Start: position, End: position},
				}, got)

				return
			}

			assert.Equal(t, protocol.Location{URI: uri.File(filepath.Join(tmpDir, tt.target))}, got)
		})
	}
}
