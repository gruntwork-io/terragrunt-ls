package hover_test

import (
	"terragrunt-ls/internal/ast"
	"terragrunt-ls/internal/testutils"
	"terragrunt-ls/internal/tg/hover"
	"terragrunt-ls/internal/tg/store"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func TestGetHoverTargetWithContext(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name            string
		store           store.Store
		position        protocol.Position
		expectedTarget  string
		expectedContext string
	}{
		{
			name:            "empty document",
			store:           store.Store{},
			position:        protocol.Position{Line: 0, Character: 0},
			expectedContext: "null",
		},
		{
			name:            "local variable",
			store:           store.Store{Document: "foo = local.var"},
			position:        protocol.Position{Line: 0, Character: 6},
			expectedTarget:  "var",
			expectedContext: "local",
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := tt.store

			fileAST, err := ast.IndexFileAST("test.hcl", []byte(store.Document))
			require.NoError(t, err)

			store.AST = fileAST

			l := testutils.NewTestLogger(t)

			target, context := hover.GetHoverTargetWithContext(l, store, tt.position)

			assert.Equal(t, tt.expectedTarget, target)
			assert.Equal(t, tt.expectedContext, context)
		})
	}
}
