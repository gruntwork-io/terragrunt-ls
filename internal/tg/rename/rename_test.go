package rename_test

import (
	"terragrunt-ls/internal/testutils"
	"terragrunt-ls/internal/tg/rename"
	"terragrunt-ls/internal/tg/store"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
)

func TestGetRenameTargetWithContext(t *testing.T) {
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
			expectedTarget:  "",
			expectedContext: "null",
		},
		{
			name:            "local variable reference",
			store:           store.Store{Document: "local.my_var"},
			position:        protocol.Position{Line: 0, Character: 6},
			expectedTarget:  "my_var",
			expectedContext: "local",
		},
		{
			name:            "local variable reference at start",
			store:           store.Store{Document: "local.my_var"},
			position:        protocol.Position{Line: 0, Character: 0},
			expectedTarget:  "my_var",
			expectedContext: "local",
		},
		{
			name:            "local variable test (from user bug report)",
			store:           store.Store{Document: "local.test"},
			position:        protocol.Position{Line: 0, Character: 6},
			expectedTarget:  "test",
			expectedContext: "local",
		},
		{
			name:            "not a renameable identifier",
			store:           store.Store{Document: "include {}"},
			position:        protocol.Position{Line: 0, Character: 0},
			expectedTarget:  "",
			expectedContext: "null",
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l := testutils.NewTestLogger(t)

			target, context := rename.GetRenameTargetWithContext(l, tt.store, tt.position)

			assert.Equal(t, tt.expectedTarget, target)
			assert.Equal(t, tt.expectedContext, context)
		})
	}
}

func TestFindAllOccurrences(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name           string
		document       string
		identifier     string
		context        string
		expectedRanges []protocol.Range
	}{
		{
			name:       "single local variable reference",
			document:   "dependency \"test\" {\n  config_path = local.my_var\n}",
			identifier: "my_var",
			context:    "local",
			expectedRanges: []protocol.Range{
				{
					Start: protocol.Position{Line: 1, Character: 22},
					End:   protocol.Position{Line: 1, Character: 28},
				},
			},
		},
		{
			name: "local variable definition and multiple references",
			document: `locals {
  my_var = "value"
}

dependency "test" {
  config_path = local.my_var
}

dependency "test2" {
  config_path = local.my_var
}`,
			identifier: "my_var",
			context:    "local",
			expectedRanges: []protocol.Range{
				// Definition in locals block
				{
					Start: protocol.Position{Line: 1, Character: 2},
					End:   protocol.Position{Line: 1, Character: 8},
				},
				// First reference
				{
					Start: protocol.Position{Line: 5, Character: 22},
					End:   protocol.Position{Line: 5, Character: 28},
				},
				// Second reference
				{
					Start: protocol.Position{Line: 9, Character: 22},
					End:   protocol.Position{Line: 9, Character: 28},
				},
			},
		},
		{
			name: "variable in string should not be matched",
			document: `locals {
  my_var = "my_var is a string"
}

dependency "test" {
  config_path = local.my_var
}`,
			identifier: "my_var",
			context:    "local",
			expectedRanges: []protocol.Range{
				// Definition
				{
					Start: protocol.Position{Line: 1, Character: 2},
					End:   protocol.Position{Line: 1, Character: 8},
				},
				// Reference (not the one in the string)
				{
					Start: protocol.Position{Line: 5, Character: 22},
					End:   protocol.Position{Line: 5, Character: 28},
				},
			},
		},
		{
			name:           "no occurrences",
			document:       "locals {\n  other_var = \"value\"\n}",
			identifier:     "my_var",
			context:        "local",
			expectedRanges: []protocol.Range{},
		},
		{
			name: "identifier in string value should not be matched",
			document: `locals {
  identifier = "identifier"
}

dependency "test" {
  config_path = local.identifier
}`,
			identifier: "identifier",
			context:    "local",
			expectedRanges: []protocol.Range{
				// Definition in locals block (NOT the string value)
				{
					Start: protocol.Position{Line: 1, Character: 2},
					End:   protocol.Position{Line: 1, Character: 12},
				},
				// Reference
				{
					Start: protocol.Position{Line: 5, Character: 22},
					End:   protocol.Position{Line: 5, Character: 32},
				},
			},
		},
		{
			name: "test variable name same as value",
			document: `locals {
  test = "test"
}

dependency "dep" {
  config_path = local.test
}`,
			identifier: "test",
			context:    "local",
			expectedRanges: []protocol.Range{
				// Definition (NOT the string value)
				{
					Start: protocol.Position{Line: 1, Character: 2},
					End:   protocol.Position{Line: 1, Character: 6},
				},
				// Reference
				{
					Start: protocol.Position{Line: 5, Character: 22},
					End:   protocol.Position{Line: 5, Character: 26},
				},
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l := testutils.NewTestLogger(t)

			ranges := rename.FindAllOccurrences(l, tt.document, tt.identifier, tt.context)

			assert.Equal(t, len(tt.expectedRanges), len(ranges), "Number of ranges should match")

			// Compare each range
			for i, expected := range tt.expectedRanges {
				if i < len(ranges) {
					assert.Equal(t, expected.Start.Line, ranges[i].Start.Line, "Start line should match for range %d", i)
					assert.Equal(t, expected.Start.Character, ranges[i].Start.Character, "Start character should match for range %d", i)
					assert.Equal(t, expected.End.Line, ranges[i].End.Line, "End line should match for range %d", i)
					assert.Equal(t, expected.End.Character, ranges[i].End.Character, "End character should match for range %d", i)
				}
			}
		})
	}
}
