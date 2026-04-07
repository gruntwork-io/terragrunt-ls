package completion_test

import (
	"terragrunt-ls/internal/testutils"
	"terragrunt-ls/internal/tg/completion"
	"terragrunt-ls/internal/tg/store"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
)

func TestGetCompletions(t *testing.T) {
	t.Parallel()

	tc := []struct {
		store       store.Store
		name        string
		filename    string
		completions []protocol.CompletionItem
		position    protocol.Position
	}{
		{
			name:     "complete dep",
			filename: "terragrunt.hcl",
			store: store.Store{
				Document: `dep`,
			},
			position: protocol.Position{Line: 0, Character: 3},
			completions: []protocol.CompletionItem{
				{
					Label: "dependency",
					Documentation: protocol.MarkupContent{
						Kind:  protocol.Markdown,
						Value: "# dependency\nThe dependency block is used to configure unit dependencies.\nEach dependency block exposes outputs of the dependency unit as variables you can reference in dependent unit configuration.",
					},
					Kind:             protocol.CompletionItemKindClass,
					InsertTextFormat: protocol.InsertTextFormatSnippet,
					TextEdit: &protocol.TextEdit{
						Range: protocol.Range{
							Start: protocol.Position{Line: 0, Character: 0},
							End:   protocol.Position{Line: 0, Character: 3},
						},
						NewText: `dependency "${1}" {
	config_path = "${2}"
}`,
					},
				},
				{
					Label: "dependencies",
					Documentation: protocol.MarkupContent{
						Kind:  protocol.Markdown,
						Value: "# dependencies\nThe dependencies block is used to enumerate all the Terragrunt units that need to be applied before this unit.",
					},
					Kind:             protocol.CompletionItemKindClass,
					InsertTextFormat: protocol.InsertTextFormatSnippet,
					TextEdit: &protocol.TextEdit{
						Range: protocol.Range{
							Start: protocol.Position{Line: 0, Character: 0},
							End:   protocol.Position{Line: 0, Character: 3},
						},
						NewText: `dependencies {
	paths = ["${1}"]
}`,
					},
				},
			},
		},
		{
			name:     "complete dependency",
			filename: "terragrunt.hcl",
			store: store.Store{
				Document: `dependency`,
			},
			position: protocol.Position{Line: 0, Character: 3},
			completions: []protocol.CompletionItem{
				{
					Label: "dependency",
					Documentation: protocol.MarkupContent{
						Kind:  protocol.Markdown,
						Value: "# dependency\nThe dependency block is used to configure unit dependencies.\nEach dependency block exposes outputs of the dependency unit as variables you can reference in dependent unit configuration.",
					},
					Kind:             protocol.CompletionItemKindClass,
					InsertTextFormat: protocol.InsertTextFormatSnippet,
					TextEdit: &protocol.TextEdit{
						Range: protocol.Range{
							Start: protocol.Position{Line: 0, Character: 0},
							End:   protocol.Position{Line: 0, Character: 3},
						},
						NewText: `dependency "${1}" {
	config_path = "${2}"
}`,
					},
				},
			},
		},
		{
			name:     "complete include",
			filename: "terragrunt.hcl",
			store: store.Store{
				Document: `in`,
			},
			position: protocol.Position{Line: 0, Character: 1},
			completions: []protocol.CompletionItem{
				{
					Label: "include",
					Documentation: protocol.MarkupContent{
						Kind:  protocol.Markdown,
						Value: "# include\nThe include block is used to specify the inclusion of partial Terragrunt configuration.",
					},
					Kind:             protocol.CompletionItemKindClass,
					InsertTextFormat: protocol.InsertTextFormatSnippet,
					TextEdit: &protocol.TextEdit{
						Range: protocol.Range{
							Start: protocol.Position{Line: 0, Character: 0},
							End:   protocol.Position{Line: 0, Character: 1},
						},
						NewText: `include "${1:root}" {
	path = ${2:find_in_parent_folders("root.hcl")}
}`,
					},
				},
				{
					Label: "inputs",
					Documentation: protocol.MarkupContent{
						Kind:  protocol.Markdown,
						Value: "# inputs\nThe inputs attribute is a map that is used to specify the input variables and their values to pass in to OpenTofu/Terraform.",
					},
					Kind:             protocol.CompletionItemKindField,
					InsertTextFormat: protocol.InsertTextFormatSnippet,
					TextEdit: &protocol.TextEdit{
						Range: protocol.Range{
							Start: protocol.Position{Line: 0, Character: 0},
							End:   protocol.Position{Line: 0, Character: 1},
						},
						NewText: `inputs = {
	${1} = ${2}
}`,
					},
				},
			},
		},
		{
			name:     "complete include",
			filename: "terragrunt.hcl",
			store: store.Store{
				Document: `include`,
			},
			position: protocol.Position{Line: 0, Character: 3},
			completions: []protocol.CompletionItem{
				{
					Label: "include",
					Documentation: protocol.MarkupContent{
						Kind:  protocol.Markdown,
						Value: "# include\nThe include block is used to specify the inclusion of partial Terragrunt configuration.",
					},
					Kind:             protocol.CompletionItemKindClass,
					InsertTextFormat: protocol.InsertTextFormatSnippet,
					TextEdit: &protocol.TextEdit{
						Range: protocol.Range{
							Start: protocol.Position{Line: 0, Character: 0},
							End:   protocol.Position{Line: 0, Character: 3},
						},
						NewText: `include "${1:root}" {
	path = ${2:find_in_parent_folders("root.hcl")}
}`,
					},
				},
			},
		},
		{
			name:     "complete generate",
			filename: "terragrunt.hcl",
			store: store.Store{
				Document: `generate`,
			},
			position: protocol.Position{Line: 0, Character: 3},
			completions: []protocol.CompletionItem{
				{
					Label: "generate",
					Documentation: protocol.MarkupContent{
						Kind:  protocol.Markdown,
						Value: "# generate\nThe generate block can be used to arbitrarily generate a file in the terragrunt working directory.",
					},
					Kind:             protocol.CompletionItemKindClass,
					InsertTextFormat: protocol.InsertTextFormatSnippet,
					TextEdit: &protocol.TextEdit{
						Range: protocol.Range{
							Start: protocol.Position{Line: 0, Character: 0},
							End:   protocol.Position{Line: 0, Character: 3},
						},
						NewText: `generate "provider" {
  path      = "${1:provider.tf}"
  if_exists = "${2:overwrite}"
  contents = <<EOF
provider "${3:aws}" {
  region = "${4:us-east-1}"
}
EOF
}`,
					},
				},
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l := testutils.NewTestLogger(t)

			completions := completion.GetCompletions(l, tt.store, tt.position, tt.filename)

			assert.ElementsMatch(t, tt.completions, completions)
		})
	}
}

func TestGetCompletion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		filename       string
		expectedLabels []string
		position       protocol.Position
	}{
		{
			name:     "terragrunt.hcl completions",
			filename: "/path/to/terragrunt.hcl",
			position: protocol.Position{Line: 0, Character: 0},
			expectedLabels: []string{
				"locals",
				"terraform",
				"remote_state",
				"include",
				"dependencies",
				"dependency",
				"generate",
				"inputs",
			},
		},
		{
			name:     "terragrunt.stack.hcl completions",
			filename: "/path/to/terragrunt.stack.hcl",
			position: protocol.Position{Line: 0, Character: 0},
			expectedLabels: []string{
				"unit",
				"stack",
			},
		},
		{
			name:     "terragrunt.values.hcl completions",
			filename: "/path/to/terragrunt.values.hcl",
			position: protocol.Position{Line: 0, Character: 0},
			expectedLabels: []string{
				"values",
				"dependency",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l := testutils.NewTestLogger(t)
			store := store.Store{
				Document: "",
			}

			completions := completion.GetCompletions(l, store, tt.position, tt.filename)

			// Check that we got some completions
			assert.NotEmpty(t, completions, "expected to get completions")

			// Check that expected labels are present
			completionLabels := make([]string, len(completions))
			for i, completion := range completions {
				completionLabels[i] = completion.Label
			}

			for _, expectedLabel := range tt.expectedLabels {
				assert.Contains(t, completionLabels, expectedLabel,
					"expected completion label %s not found in %v", expectedLabel, completionLabels)
			}
		})
	}
}

func TestGetTerragruntFileType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		filename string
		name     string
		expected completion.TerragruntFileType
	}{
		{
			name:     "terragrunt.hcl file",
			filename: "/path/to/terragrunt.hcl",
			expected: completion.TerragruntFileTypeConfig,
		},
		{
			name:     "terragrunt.stack.hcl file",
			filename: "/path/to/terragrunt.stack.hcl",
			expected: completion.TerragruntFileTypeStack,
		},
		{
			name:     "terragrunt.values.hcl file",
			filename: "/path/to/terragrunt.values.hcl",
			expected: completion.TerragruntFileTypeValues,
		},
		{
			name:     "other .hcl file",
			filename: "/path/to/variables.hcl",
			expected: completion.TerragruntFileTypeConfig,
		},
		{
			name:     "non-hcl file",
			filename: "/path/to/main.tf",
			expected: completion.TerragruntFileTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := completion.GetTerragruntFileType(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}
