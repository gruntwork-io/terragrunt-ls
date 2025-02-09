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
		name        string
		store       store.Store
		position    protocol.Position
		completions []protocol.CompletionItem
	}{
		{
			name:     "empty document",
			store:    store.Store{},
			position: protocol.Position{Line: 0, Character: 0},
			completions: []protocol.CompletionItem{
				{Label: "dependency"},
				{Label: "inputs"},
				{Label: "local"},
				{Label: "locals"},
				{Label: "feature"},
				{Label: "terraform"},
				{Label: "remote_state"},
				{Label: "include"},
				{Label: "dependencies"},
				{Label: "generate"},
				{Label: "engine"},
				{Label: "exclude"},
				{Label: "download_dir"},
				{Label: "prevent_destroy"},
				{Label: "iam_role"},
				{Label: "iam_assume_role_duration"},
				{Label: "iam_assume_role_session_name"},
				{Label: "iam_web_identity_token"},
				{Label: "terraform_binary"},
				{Label: "terraform_version_constraint"},
				{Label: "terragrunt_version_constraint"},
			},
		},
		{
			name: "complete dep",
			store: store.Store{
				Document: `dep`,
			},
			position: protocol.Position{Line: 0, Character: 2},
			completions: []protocol.CompletionItem{
				{Label: "dependency"},
				{Label: "dependencies"},
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l := testutils.NewTestLogger(t)

			completions := completion.GetCompletions(l, tt.store, tt.position)

			assert.ElementsMatch(t, tt.completions, completions)
		})
	}
}
