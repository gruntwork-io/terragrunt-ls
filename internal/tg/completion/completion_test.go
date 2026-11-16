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
		completions []protocol.CompletionItem
		position    protocol.Position
	}{
		{
			name: "complete dep",
			store: store.Store{
				Document: `dep`,
				FileType: store.FileTypeUnit,
			},
			position: protocol.Position{Line: 0, Character: 3},
			completions: []protocol.CompletionItem{
				{
					Label: "dependency",
					Documentation: &protocol.MarkupContent{
						Kind:  protocol.MarkupKindMarkdown,
						Value: testutils.DocsBlurb("Reads the outputs of another unit.", "blocks/dependency"),
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
					Documentation: &protocol.MarkupContent{
						Kind:  protocol.MarkupKindMarkdown,
						Value: testutils.DocsBlurb("Lists the units to run before this one.", "blocks/dependencies"),
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
			name: "complete dependency",
			store: store.Store{
				Document: `dependency`,
				FileType: store.FileTypeUnit,
			},
			position: protocol.Position{Line: 0, Character: 3},
			completions: []protocol.CompletionItem{
				{
					Label: "dependency",
					Documentation: &protocol.MarkupContent{
						Kind:  protocol.MarkupKindMarkdown,
						Value: testutils.DocsBlurb("Reads the outputs of another unit.", "blocks/dependency"),
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
			name: "complete include",
			store: store.Store{
				Document: `in`,
				FileType: store.FileTypeUnit,
			},
			position: protocol.Position{Line: 0, Character: 1},
			completions: []protocol.CompletionItem{
				{
					Label: "include",
					Documentation: &protocol.MarkupContent{
						Kind:  protocol.MarkupKindMarkdown,
						Value: testutils.DocsBlurb("Merges another configuration file into this one.", "blocks/include"),
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
					Documentation: &protocol.MarkupContent{
						Kind:  protocol.MarkupKindMarkdown,
						Value: testutils.DocsBlurb("Sets the input variables of the module.", "attributes/inputs"),
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
			name: "complete include",
			store: store.Store{
				Document: `include`,
				FileType: store.FileTypeUnit,
			},
			position: protocol.Position{Line: 0, Character: 3},
			completions: []protocol.CompletionItem{
				{
					Label: "include",
					Documentation: &protocol.MarkupContent{
						Kind:  protocol.MarkupKindMarkdown,
						Value: testutils.DocsBlurb("Merges another configuration file into this one.", "blocks/include"),
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
			name: "complete generate",
			store: store.Store{
				Document: `generate`,
				FileType: store.FileTypeUnit,
			},
			position: protocol.Position{Line: 0, Character: 3},
			completions: []protocol.CompletionItem{
				{
					Label: "generate",
					Documentation: &protocol.MarkupContent{
						Kind:  protocol.MarkupKindMarkdown,
						Value: testutils.DocsBlurb("Writes a file into the working directory.", "blocks/generate"),
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
		{
			name: "stack file - complete unit",
			store: store.Store{
				Document: `uni`,
				FileType: store.FileTypeStack,
			},
			position: protocol.Position{Line: 0, Character: 3},
			completions: []protocol.CompletionItem{
				{
					Label: "unit",
					Documentation: &protocol.MarkupContent{
						Kind:  protocol.MarkupKindMarkdown,
						Value: testutils.DocsBlurb("Defines a unit.", "blocks/unit"),
					},
					Kind:             protocol.CompletionItemKindClass,
					InsertTextFormat: protocol.InsertTextFormatSnippet,
					TextEdit: &protocol.TextEdit{
						Range: protocol.Range{
							Start: protocol.Position{Line: 0, Character: 0},
							End:   protocol.Position{Line: 0, Character: 3},
						},
						NewText: `unit "${1}" {
	source = "${2}"
	path   = "${3}"
}`,
					},
				},
			},
		},
		{
			name: "stack file - complete stack",
			store: store.Store{
				Document: `sta`,
				FileType: store.FileTypeStack,
			},
			position: protocol.Position{Line: 0, Character: 3},
			completions: []protocol.CompletionItem{
				{
					Label: "stack",
					Documentation: &protocol.MarkupContent{
						Kind:  protocol.MarkupKindMarkdown,
						Value: testutils.DocsBlurb("Defines a nested stack.", "blocks/stack"),
					},
					Kind:             protocol.CompletionItemKindClass,
					InsertTextFormat: protocol.InsertTextFormatSnippet,
					TextEdit: &protocol.TextEdit{
						Range: protocol.Range{
							Start: protocol.Position{Line: 0, Character: 0},
							End:   protocol.Position{Line: 0, Character: 3},
						},
						NewText: `stack "${1}" {
	source = "${2}"
	path   = "${3}"
}`,
					},
				},
			},
		},
		{
			name: "stack file - complete locals",
			store: store.Store{
				Document: `lo`,
				FileType: store.FileTypeStack,
			},
			position: protocol.Position{Line: 0, Character: 2},
			completions: []protocol.CompletionItem{
				{
					Label: "locals",
					Documentation: &protocol.MarkupContent{
						Kind:  protocol.MarkupKindMarkdown,
						Value: testutils.DocsBlurb("Defines values to reuse in the file.", "blocks/locals"),
					},
					Kind:             protocol.CompletionItemKindClass,
					InsertTextFormat: protocol.InsertTextFormatSnippet,
					TextEdit: &protocol.TextEdit{
						Range: protocol.Range{
							Start: protocol.Position{Line: 0, Character: 0},
							End:   protocol.Position{Line: 0, Character: 2},
						},
						NewText: `locals {
	${1} = ${2}
}`,
					},
				},
			},
		},
		{
			name: "values file - no completions",
			store: store.Store{
				Document: `loc`,
				FileType: store.FileTypeValues,
			},
			position:    protocol.Position{Line: 0, Character: 3},
			completions: []protocol.CompletionItem{},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l := testutils.NewTestLogger(t)

			completions := completion.GetCompletions(l, testutils.NewDocs(t), &tt.store, tt.position)

			assert.ElementsMatch(t, tt.completions, completions)
		})
	}
}
