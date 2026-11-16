// Package completion provides the logic for providing completions to the LSP client.
package completion

import (
	"strings"
	"terragrunt-ls/internal/docs"
	"terragrunt-ls/internal/logger"
	"terragrunt-ls/internal/tg/store"
	"terragrunt-ls/internal/tg/text"

	"go.lsp.dev/protocol"
)

// candidate is a block or attribute that completion offers, and the snippet it
// inserts.
type candidate struct {
	label   string
	snippet string
	kind    docs.Kind
}

// unitCandidates are the completions for terragrunt.hcl files.
//
// TODO: Add detection via the AST index to determine
// whether the cursor is in the context of a block or expression.
var unitCandidates = []candidate{
	{label: "dependency", kind: docs.KindBlock, snippet: `dependency "${1}" {
	config_path = "${2}"
}`},
	{label: "inputs", kind: docs.KindAttribute, snippet: `inputs = {
	${1} = ${2}
}`},
	{label: "locals", kind: docs.KindBlock, snippet: `locals {
	${1} = ${2}
}`},
	{label: "feature", kind: docs.KindBlock, snippet: `feature "${1}" {
	default = ${2}
}`},
	{label: "terraform", kind: docs.KindBlock, snippet: `terraform {
	source = "${1}"
}`},
	{label: "remote_state", kind: docs.KindBlock, snippet: `remote_state {
	backend = "${1:s3}"
	config = {
		bucket = "${2}"
		key = "${3}"
		region = "${4}"
	}
}`},
	{label: "include", kind: docs.KindBlock, snippet: `include "${1:root}" {
	path = ${2:find_in_parent_folders("root.hcl")}
}`},
	{label: "dependencies", kind: docs.KindBlock, snippet: `dependencies {
	paths = ["${1}"]
}`},
	{label: "generate", kind: docs.KindBlock, snippet: `generate "provider" {
  path      = "${1:provider.tf}"
  if_exists = "${2:overwrite}"
  contents = <<EOF
provider "${3:aws}" {
  region = "${4:us-east-1}"
}
EOF
}`},
	{label: "engine", kind: docs.KindBlock, snippet: `engine {
  source  = "${1:github.com/gruntwork-io/terragrunt-engine-opentofu}"
  version = "${2:v0.0.16}"
}`},
	{label: "exclude", kind: docs.KindBlock, snippet: `exclude {
	if      = ${1:true}
	actions = ["${2:all}"]
}`},
	{label: "download_dir", kind: docs.KindAttribute, snippet: `download_dir = "${1}"`},
	{label: "prevent_destroy", kind: docs.KindAttribute, snippet: `prevent_destroy = ${1:true}`},
	{label: "iam_role", kind: docs.KindAttribute, snippet: `iam_role = "arn:aws:iam::${1}:role/${2}"`},
	{label: "iam_assume_role_duration", kind: docs.KindAttribute, snippet: `iam_assume_role_duration = ${1:3600}`},
	{label: "iam_assume_role_session_name", kind: docs.KindAttribute, snippet: `iam_assume_role_session_name = "${1}"`},
	{label: "iam_web_identity_token", kind: docs.KindAttribute, snippet: `iam_web_identity_token = ${1}`},
	{label: "terraform_binary", kind: docs.KindAttribute, snippet: `terraform_binary = "${1}"`},
	{
		label:   "terraform_version_constraint",
		kind:    docs.KindAttribute,
		snippet: `terraform_version_constraint = ">= ${1:0.11}"`,
	},
	{
		label:   "terragrunt_version_constraint",
		kind:    docs.KindAttribute,
		snippet: `terragrunt_version_constraint = ">= ${1:0.23}"`,
	},
}

// stackCandidates are the completions for terragrunt.stack.hcl files.
var stackCandidates = []candidate{
	{label: "unit", kind: docs.KindBlock, snippet: `unit "${1}" {
	source = "${2}"
	path   = "${3}"
}`},
	{label: "stack", kind: docs.KindBlock, snippet: `stack "${1}" {
	source = "${2}"
	path   = "${3}"
}`},
	{label: "locals", kind: docs.KindBlock, snippet: `locals {
	${1} = ${2}
}`},
}

// GetCompletions returns the blocks and attributes valid in the document that
// start with the word under the cursor, each with its description from d.
func GetCompletions(
	l logger.Logger,
	d *docs.Docs,
	s *store.Store,
	position protocol.Position,
) []protocol.CompletionItem {
	word := text.GetCursorWord(s.Document, position)

	var candidates []candidate

	switch s.FileType {
	case store.FileTypeUnit:
		candidates = unitCandidates
	case store.FileTypeStack:
		candidates = stackCandidates
	case store.FileTypeValues, store.FileTypeUnknown:
		return []protocol.CompletionItem{}
	}

	cfg, _ := s.FileType.DocsConfig()

	completions := []protocol.CompletionItem{}

	for _, c := range candidates {
		if !strings.HasPrefix(c.label, word) {
			continue
		}

		completions = append(completions, completionItem(d, cfg, c, position))
	}

	return completions
}

func completionItem(d *docs.Docs, cfg docs.Config, c candidate, position protocol.Position) protocol.CompletionItem {
	kind := protocol.CompletionItemKindField
	if c.kind == docs.KindBlock {
		kind = protocol.CompletionItemKindClass
	}

	item := protocol.CompletionItem{
		Label:            c.label,
		Kind:             kind,
		InsertTextFormat: protocol.InsertTextFormatSnippet,
		TextEdit: &protocol.TextEdit{
			Range: protocol.Range{
				Start: protocol.Position{Line: position.Line, Character: 0},
				End:   protocol.Position{Line: position.Line, Character: position.Character},
			},
			NewText: c.snippet,
		},
	}

	if blurb, ok := d.Blurb(cfg, c.kind, c.label); ok {
		item.Documentation = &protocol.MarkupContent{Kind: protocol.MarkupKindMarkdown, Value: blurb}
	}

	return item
}
