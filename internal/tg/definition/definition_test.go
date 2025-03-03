package definition_test

import (
	"terragrunt-ls/internal/testutils"
	"terragrunt-ls/internal/tg/definition"
	"terragrunt-ls/internal/tg/store"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
)

func TestGetDefinitionTargetWithContext(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name            string
		store           store.Store
		position        protocol.Position
		expectedTarget  string
		expectedContext string
	}{
		{
			name:            "empty store",
			store:           store.Store{},
			position:        protocol.Position{Line: 0, Character: 0},
			expectedTarget:  "",
			expectedContext: "null",
		},
		{
			name: "include definition",
			store: store.Store{
				Document: `include "root" {
	path = find_in_parent_folders("root")
}`,
			},
			position:        protocol.Position{Line: 0, Character: 0},
			expectedTarget:  "root",
			expectedContext: "include",
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l := testutils.NewTestLogger(t)

			target, context := definition.GetDefinitionTargetWithContext(l, tt.store, tt.position)

			assert.Equal(t, tt.expectedTarget, target)
			assert.Equal(t, tt.expectedContext, context)
		})
	}
}
