package tg_test

import (
	"terragrunt-ls/internal/testutils"
	"terragrunt-ls/internal/tg"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func TestTextDocumentRename_LocalVariable(t *testing.T) {
	t.Parallel()

	document := `locals {
  test = "value"
}

dependency "dep1" {
  config_path = local.test
}

dependency "dep2" {
  config_path = local.test
}`

	state := tg.NewState()
	l := testutils.NewTestLogger(t)
	docURI := uri.File("/test/terragrunt.hcl")

	// Open the document
	state.OpenDocument(l, docURI, document)

	t.Run("rename with local. prefix", func(t *testing.T) {
		// Rename local.test to local.test2 (position on "test" in local.test reference)
		position := protocol.Position{Line: 5, Character: 22} // On "test" in "local.test"
		response := state.TextDocumentRename(l, 1, docURI, position, "local.test2")

		require.NotNil(t, response.Result, "Response should contain workspace edit")
		require.NotNil(t, response.Result.Changes, "WorkspaceEdit should have changes")

		edits := response.Result.Changes[docURI]
		require.Len(t, edits, 3, "Should have 3 edits (1 definition + 2 references)")

		// Verify all edits use "test2" not "local.test2"
		for _, edit := range edits {
			assert.Equal(t, "test2", edit.NewText, "New text should be just 'test2', not 'local.test2'")
		}

		// Check specific ranges
		// Definition in locals block
		assert.Equal(t, uint32(1), edits[0].Range.Start.Line)
		assert.Equal(t, uint32(2), edits[0].Range.Start.Character)
		assert.Equal(t, uint32(1), edits[0].Range.End.Line)
		assert.Equal(t, uint32(6), edits[0].Range.End.Character)

		// First reference
		assert.Equal(t, uint32(5), edits[1].Range.Start.Line)
		assert.Equal(t, uint32(22), edits[1].Range.Start.Character)
		assert.Equal(t, uint32(5), edits[1].Range.End.Line)
		assert.Equal(t, uint32(26), edits[1].Range.End.Character)

		// Second reference
		assert.Equal(t, uint32(9), edits[2].Range.Start.Line)
		assert.Equal(t, uint32(22), edits[2].Range.Start.Character)
		assert.Equal(t, uint32(9), edits[2].Range.End.Line)
		assert.Equal(t, uint32(26), edits[2].Range.End.Character)
	})

	t.Run("rename without local. prefix", func(t *testing.T) {
		// Rename local.test to just "test2" (without local. prefix)
		position := protocol.Position{Line: 5, Character: 22}
		response := state.TextDocumentRename(l, 2, docURI, position, "test2")

		require.NotNil(t, response.Result)
		edits := response.Result.Changes[docURI]
		require.Len(t, edits, 3)

		// All edits should use "test2"
		for _, edit := range edits {
			assert.Equal(t, "test2", edit.NewText)
		}
	})

	t.Run("rename from definition", func(t *testing.T) {
		// Rename from the definition in locals block
		position := protocol.Position{Line: 1, Character: 4} // On "test" in definition
		response := state.TextDocumentRename(l, 3, docURI, position, "test2")

		require.NotNil(t, response.Result)
		edits := response.Result.Changes[docURI]
		require.Len(t, edits, 3, "Should rename all occurrences when renaming from definition")
	})

	t.Run("rename invalid position", func(t *testing.T) {
		// Try to rename at an invalid position (whitespace)
		position := protocol.Position{Line: 0, Character: 0}
		response := state.TextDocumentRename(l, 4, docURI, position, "test2")

		assert.Nil(t, response.Result, "Should return nil for invalid rename position")
	})
}

func TestTextDocumentRename_IdentifierInStringValue(t *testing.T) {
	t.Parallel()

	// Test case where the identifier name appears in the string value
	document := `locals {
  identifier = "identifier"
}

dependency "test" {
  config_path = local.identifier
}`

	state := tg.NewState()
	l := testutils.NewTestLogger(t)
	docURI := uri.File("/test/terragrunt.hcl")

	state.OpenDocument(l, docURI, document)

	// Rename identifier to newname
	position := protocol.Position{Line: 5, Character: 22} // On "identifier" in local.identifier
	response := state.TextDocumentRename(l, 1, docURI, position, "newname")

	require.NotNil(t, response.Result)
	edits := response.Result.Changes[docURI]

	// Should only rename the identifier, not the string value
	require.Len(t, edits, 2, "Should have 2 edits (definition + reference), NOT the string value")

	// Verify the definition is renamed (line 1, not the string on same line)
	assert.Equal(t, uint32(1), edits[0].Range.Start.Line)
	assert.Equal(t, uint32(2), edits[0].Range.Start.Character)
	assert.Equal(t, "newname", edits[0].NewText)

	// Verify the reference is renamed
	assert.Equal(t, uint32(5), edits[1].Range.Start.Line)
	assert.Equal(t, uint32(22), edits[1].Range.Start.Character)
	assert.Equal(t, "newname", edits[1].NewText)

	// The string value "identifier" should NOT be in the edits
	for _, edit := range edits {
		// Make sure none of the edits touch the string value position
		// String "identifier" is at Line 1, Character 15-25 (inside quotes)
		if edit.Range.Start.Line == 1 {
			assert.Less(t, edit.Range.Start.Character, uint32(15),
				"Edit should not overlap with string value starting at char 15")
		}
	}
}
