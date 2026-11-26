package completion_test

import (
	"strings"
	"terragrunt-ls/internal/ast"
	"terragrunt-ls/internal/testutils"
	"terragrunt-ls/internal/tg/completion"
	"terragrunt-ls/internal/tg/store"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestGetCompletionsVariableFields(t *testing.T) {
	t.Parallel()

	fields := map[string]string{
		"path.download": "The directory in which OpenTofu/Terraform runs.",
		"path.include":  "The included file.",
		"path.read":     "The file that `read_terragrunt_config` reads.",
		"path.stack":    "The directory of the stack file.",
		"path.unit":     "The directory of the unit.",
	}

	tc := []struct {
		name     string
		document string
		labels   []string
		fileType store.FileType
		position protocol.Position
		start    uint32
		end      uint32
	}{
		{
			name:     "unit inputs",
			document: "inputs = {\n  dir = path.\n}\n",
			fileType: store.FileTypeUnit,
			position: protocol.Position{Line: 1, Character: 13},
			start:    8,
			end:      13,
			labels:   []string{"path.download", "path.unit"},
		},
		{
			name:     "unit locals",
			document: "locals {\n  dir = path.\n}\n",
			fileType: store.FileTypeUnit,
			position: protocol.Position{Line: 1, Character: 13},
			start:    8,
			end:      13,
			labels:   []string{"path.unit"},
		},
		{
			name:     "unit interpolation",
			document: "inputs = {\n  dir = \"${path.u\"\n}\n",
			fileType: store.FileTypeUnit,
			position: protocol.Position{Line: 1, Character: 17},
			start:    11,
			end:      17,
			labels:   []string{"path.unit"},
		},
		{
			name:     "stack locals",
			document: "locals {\n  dir = path.\n}\n",
			fileType: store.FileTypeStack,
			position: protocol.Position{Line: 1, Character: 13},
			start:    8,
			end:      13,
			labels:   []string{"path.stack"},
		},
		{
			name:     "included file inputs",
			document: "inputs = {\n  dir = path.\n}\n",
			fileType: store.FileTypeUnknown,
			position: protocol.Position{Line: 1, Character: 13},
			start:    8,
			end:      13,
			labels:   []string{"path.download", "path.include", "path.read", "path.stack", "path.unit"},
		},
		{
			name:     "included file locals",
			document: "locals {\n  dir = path.\n}\n",
			fileType: store.FileTypeUnknown,
			position: protocol.Position{Line: 1, Character: 13},
			start:    8,
			end:      13,
			labels:   []string{"path.include", "path.read", "path.stack", "path.unit"},
		},
		{
			name:     "shared file inputs",
			document: "inputs = {\n  dir = path.\n}\n",
			fileType: store.FileTypeShared,
			position: protocol.Position{Line: 1, Character: 13},
			start:    8,
			end:      13,
			labels:   []string{"path.download", "path.include", "path.read", "path.stack", "path.unit"},
		},
		{
			name:     "shared file locals",
			document: "locals {\n  dir = path.\n}\n",
			fileType: store.FileTypeShared,
			position: protocol.Position{Line: 1, Character: 13},
			start:    8,
			end:      13,
			labels:   []string{"path.include", "path.read", "path.stack", "path.unit"},
		},
		{
			name:     "unit field under the cursor",
			document: "inputs = {\n  dir = path.unit\n}\n",
			fileType: store.FileTypeUnit,
			position: protocol.Position{Line: 1, Character: 14},
			start:    8,
			end:      17,
			labels:   []string{"path.unit"},
		},
		{
			name:     "unit field before a dot",
			document: "inputs = {\n  dir = path.unit.x\n}\n",
			fileType: store.FileTypeUnit,
			position: protocol.Position{Line: 1, Character: 14},
			start:    8,
			end:      17,
			labels:   []string{"path.unit"},
		},
		{
			name:     "unit locals at the end of the file",
			document: "locals {\n  dir = path.",
			fileType: store.FileTypeUnit,
			position: protocol.Position{Line: 1, Character: 13},
			start:    8,
			end:      13,
			labels:   []string{"path.unit"},
		},
		{
			name:     "collection of a for expression",
			document: "inputs = {\n  dirs = [for path in [path.] : path]\n}\n",
			fileType: store.FileTypeUnit,
			position: protocol.Position{Line: 1, Character: 28},
			start:    23,
			end:      28,
			labels:   []string{"path.download", "path.unit"},
		},
		{
			name:     "for variable",
			document: "inputs = {\n  dirs = [for path in [{}] : path.]\n}\n",
			fileType: store.FileTypeUnit,
			position: protocol.Position{Line: 1, Character: 34},
		},
		{
			name:     "for variable at the end of the file",
			document: "inputs = {\n  dirs = [for path in [{}] : path.",
			fileType: store.FileTypeUnit,
			position: protocol.Position{Line: 1, Character: 34},
		},
		{
			name:     "for variable after characters outside the Basic Multilingual Plane",
			document: "inputs = { xs = [\"\U0001F600\U0001F600\U0001F600\", [for path in [{}] : path.]] }",
			fileType: store.FileTypeUnit,
			position: protocol.Position{Line: 0, Character: 52},
		},
		{
			name:     "unit field with a dash under the cursor",
			document: "inputs = {\n  dir = path.u-name\n}\n",
			fileType: store.FileTypeUnit,
			position: protocol.Position{Line: 1, Character: 14},
			start:    8,
			end:      19,
			labels:   []string{"path.unit"},
		},
		{
			name:     "identifier with a dash that ends with path",
			document: "inputs = {\n  x = dependency.base-path.\n}\n",
			fileType: store.FileTypeUnit,
			position: protocol.Position{Line: 1, Character: 27},
		},
		{
			name:     "attribute named path after a dot and a space",
			document: "inputs = {\n  x = local. path.\n}\n",
			fileType: store.FileTypeUnit,
			position: protocol.Position{Line: 1, Character: 18},
		},
		{
			name:     "attribute named path after a dot and a comment",
			document: "inputs = {\n  x = local./* c */path.\n}\n",
			fileType: store.FileTypeUnit,
			position: protocol.Position{Line: 1, Character: 24},
		},
		{
			name:     "attribute named path after a dot on the line before",
			document: "inputs = {\n  x = [local.\n    path.]\n}\n",
			fileType: store.FileTypeUnit,
			position: protocol.Position{Line: 2, Character: 9},
		},
		{
			name:     "values file",
			document: "dir = path.\n",
			fileType: store.FileTypeValues,
			position: protocol.Position{Line: 0, Character: 11},
		},
		{
			name:     "attribute of a local",
			document: "inputs = {\n  dir = local.\n}\n",
			fileType: store.FileTypeUnit,
			position: protocol.Position{Line: 1, Character: 14},
		},
		{
			name:     "string literal",
			document: "inputs = {\n  dir = \"../path.\"\n}\n",
			fileType: store.FileTypeUnit,
			position: protocol.Position{Line: 1, Character: 17},
		},
		{
			name:     "comment",
			document: "# path.\ninputs = {}\n",
			fileType: store.FileTypeUnit,
			position: protocol.Position{Line: 0, Character: 7},
		},
		{
			name:     "block comment",
			document: "/* see path. */\ninputs = {}\n",
			fileType: store.FileTypeUnit,
			position: protocol.Position{Line: 0, Character: 12},
		},
		{
			name:     "block comment of two lines",
			document: "/*\n  path.\n*/\ninputs = {}\n",
			fileType: store.FileTypeUnit,
			position: protocol.Position{Line: 1, Character: 7},
		},
		{
			name:     "heredoc",
			document: "inputs = {\n  dir = <<EOF\npath.\nEOF\n}\n",
			fileType: store.FileTypeUnit,
			position: protocol.Position{Line: 2, Character: 5},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			expected := []protocol.CompletionItem{}

			for _, label := range tt.labels {
				expected = append(expected, protocol.CompletionItem{
					Label:  label,
					Kind:   protocol.CompletionItemKindVariable,
					Detail: protocol.NewOptional("Experimental"),
					Documentation: &protocol.MarkupContent{
						Kind: protocol.MarkupKindMarkdown,
						Value: testutils.DocsExperimentBlurb(
							fields[label],
							"variables/"+strings.ReplaceAll(label, ".", "/"),
							"path-variable",
						),
					},
					InsertTextFormat: protocol.InsertTextFormatPlainText,
					TextEdit: &protocol.TextEdit{
						Range: protocol.Range{
							Start: protocol.Position{Line: tt.position.Line, Character: tt.start},
							End:   protocol.Position{Line: tt.position.Line, Character: tt.end},
						},
						NewText: label,
					},
				})
			}

			indexed, err := ast.ParseHCLFile("terragrunt.hcl", []byte(tt.document))
			require.NotNil(t, indexed, err)

			st := &store.Store{AST: indexed, Document: tt.document, FileType: tt.fileType}
			completions := completion.GetCompletions(testutils.NewTestLogger(t), testutils.NewDocs(t), st, tt.position)
			assert.Equal(t, expected, completions)
		})
	}
}
