// Package completion provides the logic for providing completions to the LSP client.
package completion

import (
	"slices"
	"strings"
	"terragrunt-ls/internal/ast"
	"terragrunt-ls/internal/docs"
	"terragrunt-ls/internal/logger"
	"terragrunt-ls/internal/tg/store"
	"terragrunt-ls/internal/tg/text"
	"unicode/utf16"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
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

// variableFields are the variable fields that completion offers in each file type.
var variableFields = map[store.FileType][]string{
	store.FileTypeUnit:    {"path.download", "path.unit"},
	store.FileTypeStack:   {"path.stack"},
	store.FileTypeShared:  {"path.download", "path.include", "path.read", "path.stack", "path.unit"},
	store.FileTypeUnknown: {"path.download", "path.include", "path.read", "path.stack", "path.unit"},
}

// localsExcludedFields are the variable fields that Terragrunt rejects in locals.
var localsExcludedFields = []string{"path.download"}

// GetCompletions returns the blocks and attributes valid in the document that
// start with the word under the cursor, or the variable fields before the cursor, each with its description from d.
func GetCompletions(
	l logger.Logger,
	d *docs.Docs,
	s *store.Store,
	position protocol.Position,
) []protocol.CompletionItem {
	prefix, prefixRange := text.GetCursorWordPrefix(s.Document, position)
	if variable, ok := readsVariable(prefix); ok {
		return fieldCompletions(l, d, s, variable, prefix, prefixRange)
	}

	word := text.GetCursorWord(s.Document, position)

	var candidates []candidate

	switch s.FileType {
	case store.FileTypeUnit, store.FileTypeShared:
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

// readsVariable returns the variable whose field prefix reads.
func readsVariable(prefix string) (docs.Variable, bool) {
	name, _, ok := strings.Cut(prefix, ".")
	if !ok {
		return docs.Variable{}, false
	}

	return docs.LookupVariable(name)
}

// skippedTokens are the tokens that HCL skips between a dot and the name after it.
var skippedTokens = []hclsyntax.TokenType{hclsyntax.TokenNewline, hclsyntax.TokenComment}

// rootAt returns the identifier token of tokens that starts at offset and follows no dot.
func rootAt(tokens hclsyntax.Tokens, offset int) (hclsyntax.Token, bool) {
	i := slices.IndexFunc(tokens, func(token hclsyntax.Token) bool {
		return token.Type == hclsyntax.TokenIdent && token.Range.Start.Byte == offset
	})
	if i < 0 {
		return hclsyntax.Token{}, false
	}

	before := i - 1
	for before >= 0 && slices.Contains(skippedTokens, tokens[before].Type) {
		before--
	}

	if before >= 0 && tokens[before].Type == hclsyntax.TokenDot {
		return hclsyntax.Token{}, false
	}

	return tokens[i], true
}

// identifierRest returns the text from offset to the end of the identifier token of tokens that holds offset.
func identifierRest(tokens hclsyntax.Tokens, document string, offset int) string {
	for _, token := range tokens {
		if token.Type == hclsyntax.TokenIdent && token.Range.Start.Byte <= offset && offset < token.Range.End.Byte {
			return document[offset:token.Range.End.Byte]
		}
	}

	return ""
}

// inLocals reports whether pos is in a locals block of indexed.
func inLocals(indexed *ast.IndexedAST, pos hcl.Pos) bool {
	if indexed == nil {
		return false
	}

	for n := indexed.FindNodeAt(pos); n != nil; n = n.Parent {
		if ast.IsLocalsBlock(n) {
			return true
		}
	}

	return false
}

// placeholder is an identifier character that completes a reference that ends at the cursor.
const placeholder = "_"

// declaredByFor reports whether a for expression of document declares name for the reference from pos to offset.
func declaredByFor(l logger.Logger, document, name string, pos hcl.Pos, offset int) bool {
	indexed, err := ast.ParseHCLFile("", []byte(document[:offset]+placeholder+document[offset:]))
	if err != nil {
		l.Debug("Parsing the document for completion", "error", err)
	}

	for n := indexed.FindNodeAt(pos); n != nil; n = n.Parent {
		expr, ok := n.Node.(*hclsyntax.ForExpr)
		if ok && (expr.KeyVar == name || expr.ValVar == name) && !expr.CollExpr.Range().ContainsOffset(offset) {
			return true
		}
	}

	return false
}

// fieldCompletions returns the fields of variable valid in s that start with prefix, the text of prefixRange.
func fieldCompletions(
	l logger.Logger,
	d *docs.Docs,
	s *store.Store,
	variable docs.Variable,
	prefix string,
	prefixRange protocol.Range,
) []protocol.CompletionItem {
	completions := []protocol.CompletionItem{}
	offset := text.GetCursorOffset(s.Document, prefixRange.End)

	tokens, diags := hclsyntax.LexConfig([]byte(s.Document), "", hcl.InitialPos)
	if diags.HasErrors() {
		l.Debug("Lexing the document for completion", "error", diags.Error())
	}

	root, ok := rootAt(tokens, offset-len(prefix))
	if !ok || declaredByFor(l, s.Document, variable.Name, root.Range.Start, offset) {
		return completions
	}

	reference := prefixRange
	reference.End.Character += uint32(len(utf16.Encode([]rune(identifierRest(tokens, s.Document, offset)))))

	cfg, _ := s.FileType.DocsConfig()
	locals := inLocals(s.AST, root.Range.Start)

	for _, field := range variableFields[s.FileType] {
		if !strings.HasPrefix(field, prefix) || locals && slices.Contains(localsExcludedFields, field) {
			continue
		}

		item := protocol.CompletionItem{
			Label:            field,
			Kind:             protocol.CompletionItemKindVariable,
			InsertTextFormat: protocol.InsertTextFormatPlainText,
			TextEdit:         &protocol.TextEdit{Range: reference, NewText: field},
		}

		if variable.Experiment != "" {
			item.Detail = protocol.NewOptional("Experimental")
		}

		if blurb, ok := d.Blurb(cfg, docs.KindVariable, field); ok {
			item.Documentation = &protocol.MarkupContent{Kind: protocol.MarkupKindMarkdown, Value: blurb}
		}

		completions = append(completions, item)
	}

	return completions
}
