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
	cursorWord := text.GetCursorWord(store.Document, position)
	completions := []protocol.CompletionItem{}

	for _, word := range defaultCompletionWords() {
		if strings.HasPrefix(word, cursorWord) {
			completions = append(completions, protocol.CompletionItem{
				Label: word,
			})
		}
	}

	return completions
}

func defaultCompletionWords() []string {
	return []string{
		"dependency",
		"inputs",
		"local",
		"locals",
		"feature",
		"terraform",
		"remote_state",
		"include",
		"dependencies",
		"generate",
		"engine",
		"exclude",
		"download_dir",
		"prevent_destroy",
		"iam_role",
		"iam_assume_role_duration",
		"iam_assume_role_session_name",
		"iam_web_identity_token",
		"terraform_binary",
		"terraform_version_constraint",
		"terragrunt_version_constraint",
	}
}
