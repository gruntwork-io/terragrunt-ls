package completion

import (
	"log"
	"strings"
	"terragrunt-ls/tg/store"
	"terragrunt-ls/tg/text"

	"go.lsp.dev/protocol"
)

func GetCompletions(l *log.Logger, store store.Store, position protocol.Position) []protocol.CompletionItem {
	cursorWord := text.GetCursorWord(store.Document, position)
	completions := []protocol.CompletionItem{}

	for _, term := range completionTerms() {
		if strings.HasPrefix(term, cursorWord) {
			completions = append(completions, protocol.CompletionItem{
				Label: term,
			})
		}
	}

	return completions
}

func completionTerms() []string {
	return []string{
		"dependency",
		"inputs",
		"local",
		"locals",
		"feauture",
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
