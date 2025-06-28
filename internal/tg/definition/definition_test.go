package definition_test

import (
	"terragrunt-ls/internal/testutils"
	"terragrunt-ls/internal/tg"
	"terragrunt-ls/internal/tg/definition"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
)

func TestGetDefinitionTargetWithContext(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name            string
		document        string
		expectedTarget  string
		expectedContext string
		position        protocol.Position
	}{
		{
			name:            "empty store",
			document:        "",
			position:        protocol.Position{Line: 0, Character: 0},
			expectedTarget:  "",
			expectedContext: "null",
		},
		{
			name: "include definition",
			document: `include "root" {
	path = find_in_parent_folders("root")
}`,
			position:        protocol.Position{Line: 1, Character: 8},
			expectedTarget:  "root",
			expectedContext: "include",
		},
		{
			name: "dependency definition",
			document: `dependency "vpc" {
	config_path = "../vpc"
}`,
			position:        protocol.Position{Line: 1, Character: 18},
			expectedTarget:  "vpc",
			expectedContext: "dependency",
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l := testutils.NewTestLogger(t)

			s := tg.NewState()

			s.OpenDocument(l, "file:///test.hcl", tt.document)

			target, context := definition.GetDefinitionTargetWithContext(l, s.Configs["/test.hcl"], tt.position)

			assert.Equal(t, tt.expectedTarget, target)
			assert.Equal(t, tt.expectedContext, context)
		})
	}
}
