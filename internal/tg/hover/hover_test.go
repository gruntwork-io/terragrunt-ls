package hover_test

import (
	"terragrunt-ls/internal/testutils"
	"terragrunt-ls/internal/tg/hover"
	"terragrunt-ls/internal/tg/store"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
)

func TestGetHoverTargetWithContext(t *testing.T) {
	t.Parallel()

	tc := []struct {
		store           store.Store
		name            string
		expectedTarget  string
		expectedContext string
		position        protocol.Position
	}{
		{
			name:            "empty document",
			store:           store.Store{},
			position:        protocol.Position{Line: 0, Character: 0},
			expectedContext: "null",
		},
		{
			name:            "local variable",
			store:           store.Store{Document: "local.var"},
			position:        protocol.Position{Line: 0, Character: 0},
			expectedTarget:  "var",
			expectedContext: "local",
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l := testutils.NewTestLogger(t)

			target, context := hover.GetHoverTargetWithContext(l, tt.store, tt.position)

			assert.Equal(t, tt.expectedTarget, target)
			assert.Equal(t, tt.expectedContext, context)
		})
	}
}
