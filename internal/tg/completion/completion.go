// Package completion provides the logic for providing completions to the LSP client.
package completion

import (
	"strings"
	"terragrunt-ls/internal/logger"
	"terragrunt-ls/internal/tg/store"
	"terragrunt-ls/internal/tg/text"

	"go.lsp.dev/protocol"
)

func GetCompletions(l logger.Logger, store store.Store, position protocol.Position) []protocol.CompletionItem {
	word := text.GetCursorWord(store.Document, position)
	completions := []protocol.CompletionItem{}

	for _, completion := range getCompletions(position) {
		if strings.HasPrefix(completion.Label, word) {
			completions = append(completions, completion)
		}
	}

	return completions
}

func getCompletions(position protocol.Position) []protocol.CompletionItem {
	return []protocol.CompletionItem{
		{
			Label: "dependency",
		},
		{
			Label: "inputs",
		},
		{
			Label: "local",
		},
		{
			Label: "locals",
		},
		{
			Label: "feature",
		},
		{
			Label: "terraform",
		},
		{
			Label: "remote_state",
		},
		{
			Label: "include",
			Documentation: protocol.MarkupContent{
				Kind: protocol.Markdown,
				Value: `# include

The include block allows you to include partial Terragrunt configuration from another file into the current unit.
This is useful for breaking up large Terragrunt configurations into smaller, reusable pieces.`,
			},
			Kind:             protocol.CompletionItemKindClass,
			InsertTextFormat: protocol.InsertTextFormatSnippet,
			TextEdit: &protocol.TextEdit{
				Range: protocol.Range{
					Start: protocol.Position{
						Line:      position.Line,
						Character: 0,
					},
					End: protocol.Position{
						Line:      position.Line,
						Character: position.Character,
					},
				},
				NewText: `include "${1:root}" {
	path = ${2:find_in_parent_folders("root.hcl")}
}`,
			},
		},
		{
			Label: "dependencies",
		},
		{
			Label: "generate",
		},
		{
			Label: "engine",
		},
		{
			Label: "exclude",
		},
		{
			Label: "download_dir",
		},
		{
			Label: "prevent_destroy",
		},
		{
			Label: "iam_role",
		},
		{
			Label: "iam_assume_role_duration",
		},
		{
			Label: "iam_assume_role_session_name",
		},
		{
			Label: "iam_web_identity_token",
		},
		{
			Label: "terraform_binary",
		},
		{
			Label: "terraform_version_constraint",
		},
		{
			Label: "terragrunt_version_constraint",
		},
	}
}
