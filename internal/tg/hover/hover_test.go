package hover_test

import (
	"terragrunt-ls/internal/testutils"
	"terragrunt-ls/internal/tg"
	"terragrunt-ls/internal/tg/hover"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
)

func TestGetHoverTargetWithContext(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name            string
		document        string
		position        protocol.Position
		expectedTarget  string
		expectedContext string
	}{
		{
			name:            "empty document",
			position:        protocol.Position{Line: 0, Character: 0},
			expectedContext: "null",
		},
		{
			name:            "local variable",
			document:        "foo = local.var",
			position:        protocol.Position{Line: 0, Character: 6},
			expectedTarget:  "var",
			expectedContext: "local",
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l := testutils.NewTestLogger(t)

			s := tg.NewState()

			s.OpenDocument(l, "file:///test.hcl", tt.document)

			target, context := hover.GetHoverTargetWithContext(l, s.Stores["/test.hcl"], tt.position)

			assert.Equal(t, tt.expectedTarget, target)
			assert.Equal(t, tt.expectedContext, context)
		})
	}
}
