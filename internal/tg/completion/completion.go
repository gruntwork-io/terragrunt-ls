// Package completion provides the logic for providing completions to the LSP client.
package completion

import (
	"path/filepath"
	"strings"
	"terragrunt-ls/internal/logger"
	"terragrunt-ls/internal/tg/store"
	"terragrunt-ls/internal/tg/text"

	"go.lsp.dev/protocol"
)

// TerragruntFileType represents the type of Terragrunt configuration file
type TerragruntFileType int

const (
	// TerragruntFileTypeUnknown represents an unknown file type
	TerragruntFileTypeUnknown TerragruntFileType = iota
	// TerragruntFileTypeConfig represents a standard terragrunt.hcl file
	TerragruntFileTypeConfig
	// TerragruntFileTypeStack represents a terragrunt.stack.hcl file
	TerragruntFileTypeStack
	// TerragruntFileTypeValues represents a terragrunt.values.hcl file
	TerragruntFileTypeValues
)

// GetTerragruntFileType determines the type of Terragrunt file based on its name
func GetTerragruntFileType(filename string) TerragruntFileType {
	base := filepath.Base(filename)

	switch {
	case strings.HasSuffix(base, ".stack.hcl"):
		return TerragruntFileTypeStack
	case strings.HasSuffix(base, ".values.hcl"):
		return TerragruntFileTypeValues
	case base != ".terraform.lock.hcl" && strings.HasSuffix(base, ".hcl"):
		return TerragruntFileTypeConfig
	default:
		return TerragruntFileTypeUnknown
	}
}

// GetCompletions returns completion suggestions for the given position in the document
func GetCompletions(l logger.Logger, store store.Store, position protocol.Position, filename string) []protocol.CompletionItem {
	// Determine file type for context-specific completions
	fileType := GetTerragruntFileType(filename)

	switch fileType {
	case TerragruntFileTypeStack:
		return getStackCompletions(position)
	case TerragruntFileTypeValues:
		return getValuesCompletions(position)
	case TerragruntFileTypeConfig:
		return getConfigCompletions(store, position)
	case TerragruntFileTypeUnknown:
		return []protocol.CompletionItem{}
	default:
		return []protocol.CompletionItem{}
	}
}

// newCompletions returns a list of completions for the given position.
//
// TODO: Add detection via the AST index to determine
// whether the cursor is in the context of a block or expression.
func newCompletions(position protocol.Position) []protocol.CompletionItem {
	return []protocol.CompletionItem{
		{
			Label: "dependency",
			Documentation: protocol.MarkupContent{
				Kind: protocol.Markdown,
				Value: `# dependency
The dependency block is used to configure unit dependencies.
Each dependency block exposes outputs of the dependency unit as variables you can reference in dependent unit configuration.`,
			},
			Kind:             protocol.CompletionItemKindClass,
			InsertTextFormat: protocol.InsertTextFormatSnippet,
			TextEdit: &protocol.TextEdit{
				Range: protocol.Range{
					Start: protocol.Position{Line: position.Line, Character: 0},
					End:   protocol.Position{Line: position.Line, Character: position.Character},
				},
				NewText: `dependency "${1}" {
	config_path = "${2}"
}`,
			},
		},
		{
			Label: "inputs",
			Documentation: protocol.MarkupContent{
				Kind: protocol.Markdown,
				Value: `# inputs
The inputs attribute is a map that is used to specify the input variables and their values to pass in to OpenTofu/Terraform.`,
			},
			Kind:             protocol.CompletionItemKindField,
			InsertTextFormat: protocol.InsertTextFormatSnippet,
			TextEdit: &protocol.TextEdit{
				Range: protocol.Range{
					Start: protocol.Position{Line: position.Line, Character: 0},
					End:   protocol.Position{Line: position.Line, Character: position.Character},
				},
				NewText: `inputs = {
	${1} = ${2}
}`,
			},
		},
		{
			Label: "locals",
			Documentation: protocol.MarkupContent{
				Kind: protocol.Markdown,
				Value: `# locals
The locals block is used to define aliases for Terragrunt expressions that can be referenced elsewhere in configuration.`,
			},
			Kind:             protocol.CompletionItemKindClass,
			InsertTextFormat: protocol.InsertTextFormatSnippet,
			TextEdit: &protocol.TextEdit{
				Range: protocol.Range{
					Start: protocol.Position{Line: position.Line, Character: 0},
					End:   protocol.Position{Line: position.Line, Character: position.Character},
				},
				NewText: `locals {
	${1} = ${2}
}`,
			},
		},
		{
			Label: "feature",
			Documentation: protocol.MarkupContent{
				Kind: protocol.Markdown,
				Value: `# feature
The feature block is used to configure feature flags in HCL for a specific Terragrunt unit.`,
			},
			Kind:             protocol.CompletionItemKindClass,
			InsertTextFormat: protocol.InsertTextFormatSnippet,
			TextEdit: &protocol.TextEdit{
				Range: protocol.Range{
					Start: protocol.Position{Line: position.Line, Character: 0},
					End:   protocol.Position{Line: position.Line, Character: position.Character},
				},
				NewText: `feature "${1}" {
	default = ${2}
}`,
			},
		},
		{
			Label: "terraform",
			Documentation: protocol.MarkupContent{
				Kind: protocol.Markdown,
				Value: `# terraform
The terraform block is used to configure how Terragrunt will interact with OpenTofu/Terraform.`,
			},
			Kind:             protocol.CompletionItemKindClass,
			InsertTextFormat: protocol.InsertTextFormatSnippet,
			TextEdit: &protocol.TextEdit{
				Range: protocol.Range{
					Start: protocol.Position{Line: position.Line, Character: 0},
					End:   protocol.Position{Line: position.Line, Character: position.Character},
				},
				NewText: `terraform {
	source = "${1}"
}`,
			},
		},
		{
			Label: "remote_state",
			Documentation: protocol.MarkupContent{
				Kind: protocol.Markdown,
				Value: `# remote_state
The remote_state block is used to configure how Terragrunt will set up remote state configuration.`,
			},
			Kind:             protocol.CompletionItemKindClass,
			InsertTextFormat: protocol.InsertTextFormatSnippet,
			TextEdit: &protocol.TextEdit{
				Range: protocol.Range{
					Start: protocol.Position{Line: position.Line, Character: 0},
					End:   protocol.Position{Line: position.Line, Character: position.Character},
				},
				NewText: `remote_state {
	backend = "${1:s3}"
	config = {
		bucket = "${2}"
		key = "${3}"
		region = "${4}"
	}
}`,
			},
		},
		{
			Label: "include",
			Documentation: protocol.MarkupContent{
				Kind: protocol.Markdown,
				Value: `# include
The include block is used to specify the inclusion of partial Terragrunt configuration.`,
			},
			Kind:             protocol.CompletionItemKindClass,
			InsertTextFormat: protocol.InsertTextFormatSnippet,
			TextEdit: &protocol.TextEdit{
				Range: protocol.Range{
					Start: protocol.Position{Line: position.Line, Character: 0},
					End:   protocol.Position{Line: position.Line, Character: position.Character},
				},
				NewText: `include "${1:root}" {
	path = ${2:find_in_parent_folders("root.hcl")}
}`,
			},
		},
		{
			Label: "dependencies",
			Documentation: protocol.MarkupContent{
				Kind: protocol.Markdown,
				Value: `# dependencies
The dependencies block is used to enumerate all the Terragrunt units that need to be applied before this unit.`,
			},
			Kind:             protocol.CompletionItemKindClass,
			InsertTextFormat: protocol.InsertTextFormatSnippet,
			TextEdit: &protocol.TextEdit{
				Range: protocol.Range{
					Start: protocol.Position{Line: position.Line, Character: 0},
					End:   protocol.Position{Line: position.Line, Character: position.Character},
				},
				NewText: `dependencies {
	paths = ["${1}"]
}`,
			},
		},
		{
			Label: "generate",
			Documentation: protocol.MarkupContent{
				Kind: protocol.Markdown,
				Value: `# generate
The generate block can be used to arbitrarily generate a file in the terragrunt working directory.`,
			},
			Kind:             protocol.CompletionItemKindClass,
			InsertTextFormat: protocol.InsertTextFormatSnippet,
			TextEdit: &protocol.TextEdit{
				Range: protocol.Range{
					Start: protocol.Position{Line: position.Line, Character: 0},
					End:   protocol.Position{Line: position.Line, Character: position.Character},
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
		{
			Label: "engine",
			Documentation: protocol.MarkupContent{
				Kind: protocol.Markdown,
				Value: `# engine
The engine block is used to configure Terragrunt engine configuration.`,
			},
			Kind:             protocol.CompletionItemKindClass,
			InsertTextFormat: protocol.InsertTextFormatSnippet,
			TextEdit: &protocol.TextEdit{
				Range: protocol.Range{
					Start: protocol.Position{Line: position.Line, Character: 0},
					End:   protocol.Position{Line: position.Line, Character: position.Character},
				},
				NewText: `engine {
  source  = "${1:github.com/gruntwork-io/terragrunt-engine-opentofu}"
  version = "${2:v0.0.16}"
}`,
			},
		},
		{
			Label: "exclude",
			Documentation: protocol.MarkupContent{
				Kind: protocol.Markdown,
				Value: `# exclude
The exclude block provides configuration options to dynamically determine when and how a unit is excluded from the run queue.`,
			},
			Kind:             protocol.CompletionItemKindClass,
			InsertTextFormat: protocol.InsertTextFormatSnippet,
			TextEdit: &protocol.TextEdit{
				Range: protocol.Range{
					Start: protocol.Position{Line: position.Line, Character: 0},
					End:   protocol.Position{Line: position.Line, Character: position.Character},
				},
				NewText: `exclude {
	if      = ${1:true}
	actions = ["${2:all}"]
}`,
			},
		},
		{
			Label: "download_dir",
			Documentation: protocol.MarkupContent{
				Kind: protocol.Markdown,
				Value: `# download_dir
The download_dir string option can be used to override the default download directory (which is .terragrunt-cache by default).`,
			},
			Kind:             protocol.CompletionItemKindField,
			InsertTextFormat: protocol.InsertTextFormatSnippet,
			TextEdit: &protocol.TextEdit{
				Range: protocol.Range{
					Start: protocol.Position{Line: position.Line, Character: 0},
					End:   protocol.Position{Line: position.Line, Character: position.Character},
				},
				NewText: `download_dir = "${1}"`,
			},
		},
		{
			Label: "prevent_destroy",
			Documentation: protocol.MarkupContent{
				Kind: protocol.Markdown,
				Value: `# prevent_destroy
The prevent_destroy boolean attribute prevents the unit from being destroyed.`,
			},
			Kind:             protocol.CompletionItemKindField,
			InsertTextFormat: protocol.InsertTextFormatSnippet,
			TextEdit: &protocol.TextEdit{
				Range: protocol.Range{
					Start: protocol.Position{Line: position.Line, Character: 0},
					End:   protocol.Position{Line: position.Line, Character: position.Character},
				},
				NewText: `prevent_destroy = ${1:true}`,
			},
		},
		{
			Label: "iam_role",
			Documentation: protocol.MarkupContent{
				Kind: protocol.Markdown,
				Value: `# iam_role
The iam_role attribute is used to specify an IAM role that Terragrunt should assume prior to running OpenTofu/Terraform.`,
			},
			Kind:             protocol.CompletionItemKindField,
			InsertTextFormat: protocol.InsertTextFormatSnippet,
			TextEdit: &protocol.TextEdit{
				Range: protocol.Range{
					Start: protocol.Position{Line: position.Line, Character: 0},
					End:   protocol.Position{Line: position.Line, Character: position.Character},
				},
				NewText: `iam_role = "arn:aws:iam::${1}:role/${2}"`,
			},
		},
		{
			Label: "iam_assume_role_duration",
			Documentation: protocol.MarkupContent{
				Kind: protocol.Markdown,
				Value: `# iam_assume_role_duration
The iam_assume_role_duration attribute is used to specify the STS session duration, in seconds.`,
			},
			Kind:             protocol.CompletionItemKindField,
			InsertTextFormat: protocol.InsertTextFormatSnippet,
			TextEdit: &protocol.TextEdit{
				Range: protocol.Range{
					Start: protocol.Position{Line: position.Line, Character: 0},
					End:   protocol.Position{Line: position.Line, Character: position.Character},
				},
				NewText: `iam_assume_role_duration = ${1:3600}`,
			},
		},
		{
			Label: "iam_assume_role_session_name",
			Documentation: protocol.MarkupContent{
				Kind: protocol.Markdown,
				Value: `# iam_assume_role_session_name
The iam_assume_role_session_name attribute is used to specify the STS session name.`,
			},
			Kind:             protocol.CompletionItemKindField,
			InsertTextFormat: protocol.InsertTextFormatSnippet,
			TextEdit: &protocol.TextEdit{
				Range: protocol.Range{
					Start: protocol.Position{Line: position.Line, Character: 0},
					End:   protocol.Position{Line: position.Line, Character: position.Character},
				},
				NewText: `iam_assume_role_session_name = "${1}"`,
			},
		},
		{
			Label: "iam_web_identity_token",
			Documentation: protocol.MarkupContent{
				Kind: protocol.Markdown,
				Value: `# iam_web_identity_token
The iam_web_identity_token attribute is used along with iam_role to assume a role using the AssumeRoleWithWebIdentity API.`,
			},
			Kind:             protocol.CompletionItemKindField,
			InsertTextFormat: protocol.InsertTextFormatSnippet,
			TextEdit: &protocol.TextEdit{
				Range: protocol.Range{
					Start: protocol.Position{Line: position.Line, Character: 0},
					End:   protocol.Position{Line: position.Line, Character: position.Character},
				},
				NewText: `iam_web_identity_token = ${1}`,
			},
		},
		{
			Label: "terraform_binary",
			Documentation: protocol.MarkupContent{
				Kind: protocol.Markdown,
				Value: `# terraform_binary
The terraform_binary attribute is used to override the binary Terragrunt uses during runs (which is tofu by default).`,
			},
			Kind:             protocol.CompletionItemKindField,
			InsertTextFormat: protocol.InsertTextFormatSnippet,
			TextEdit: &protocol.TextEdit{
				Range: protocol.Range{
					Start: protocol.Position{Line: position.Line, Character: 0},
					End:   protocol.Position{Line: position.Line, Character: position.Character},
				},
				NewText: `terraform_binary = "${1}"`,
			},
		},
		{
			Label: "terraform_version_constraint",
			Documentation: protocol.MarkupContent{
				Kind: protocol.Markdown,
				Value: `# terraform_version_constraint
The terraform_version_constraint attribute is used to override the default minimum supported version of OpenTofu/Terraform.`,
			},
			Kind:             protocol.CompletionItemKindField,
			InsertTextFormat: protocol.InsertTextFormatSnippet,
			TextEdit: &protocol.TextEdit{
				Range: protocol.Range{
					Start: protocol.Position{Line: position.Line, Character: 0},
					End:   protocol.Position{Line: position.Line, Character: position.Character},
				},
				NewText: `terraform_version_constraint = ">= ${1:0.11}"`,
			},
		},
		{
			Label: "terragrunt_version_constraint",
			Documentation: protocol.MarkupContent{
				Kind: protocol.Markdown,
				Value: `# terragrunt_version_constraint
The terragrunt_version_constraint attribute is used to specify which versions of the Terragrunt CLI can be used.`,
			},
			Kind:             protocol.CompletionItemKindField,
			InsertTextFormat: protocol.InsertTextFormatSnippet,
			TextEdit: &protocol.TextEdit{
				Range: protocol.Range{
					Start: protocol.Position{Line: position.Line, Character: 0},
					End:   protocol.Position{Line: position.Line, Character: position.Character},
				},
				NewText: `terragrunt_version_constraint = ">= ${1:0.23}"`,
			},
		},
	}
}

// createCompletionItem creates a completion item with the given parameters
func createCompletionItem(label, docValue, newText string, position protocol.Position) protocol.CompletionItem {
	return protocol.CompletionItem{
		Label: label,
		Documentation: protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: docValue,
		},
		Kind:             protocol.CompletionItemKindClass,
		InsertTextFormat: protocol.InsertTextFormatSnippet,
		TextEdit: &protocol.TextEdit{
			Range: protocol.Range{
				Start: protocol.Position{Line: position.Line, Character: 0},
				End:   protocol.Position{Line: position.Line, Character: position.Character},
			},
			NewText: newText,
		},
	}
}

// getStackCompletions returns completions specific to terragrunt.stack.hcl files
func getStackCompletions(position protocol.Position) []protocol.CompletionItem {
	return []protocol.CompletionItem{
		createCompletionItem(
			"unit",
			`# unit
The unit block is used to define a single infrastructure unit in a Terragrunt stack.`,
			`unit "${1:name}" {
	source = "${2}"
	path   = "${3}"
}`,
			position,
		),
		createCompletionItem(
			"stack",
			`# stack
The stack block is used to define a nested stack within a Terragrunt stack.`,
			`stack "${1:name}" {
	source = "${2}"
	path   = "${3}"
}`,
			position,
		),
	}
}

// getValuesCompletions returns completions specific to terragrunt.values.hcl files
func getValuesCompletions(position protocol.Position) []protocol.CompletionItem {
	return []protocol.CompletionItem{
		createCompletionItem(
			"values",
			`# values
The values block is used to define dynamic values for units in Terragrunt stacks.`,
			`values {
	${1:key} = "${2:value}"
}`,
			position,
		),
		createCompletionItem(
			"dependency",
			`# dependency
The dependency block is used to reference outputs from other units in values files.`,
			`dependency "${1:name}" {
	config_path = "${2}"

	mock_outputs = {
		${3:output_name} = "${4:mock_value}"
	}
}`,
			position,
		),
	}
}

// getConfigCompletions returns completions for standard terragrunt.hcl files
func getConfigCompletions(store store.Store, position protocol.Position) []protocol.CompletionItem {
	word := text.GetCursorWord(store.Document, position)
	completions := []protocol.CompletionItem{}

	for _, completion := range newCompletions(position) {
		if strings.HasPrefix(completion.Label, word) {
			completions = append(completions, completion)
		}
	}

	return completions
}
