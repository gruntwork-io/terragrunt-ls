package completion_test

import (
	"strings"
	"terragrunt-ls/internal/testutils"
	"terragrunt-ls/internal/tg/completion"
	"terragrunt-ls/internal/tg/store"
	"terragrunt-ls/internal/tg/text"
	"testing"

	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

// FuzzGetCompletions tests that every completion starts with the cursor word and that values and unknown files get none.
func FuzzGetCompletions(f *testing.F) {
	fileTypes := []store.FileType{
		store.FileTypeUnknown,
		store.FileTypeUnit,
		store.FileTypeStack,
		store.FileTypeValues,
	}

	f.Add("dep", uint32(0), uint32(3), uint8(1))
	f.Add("un", uint32(0), uint32(2), uint8(2))
	f.Add("", uint32(0), uint32(0), uint8(1))
	// Regression: a character past the end of the line panicked.
	f.Add("0", uint32(0), uint32(86), uint8(1))
	f.Add("0", uint32(0), uint32(86), uint8(3))

	f.Fuzz(func(t *testing.T, document string, line, character uint32, fileType uint8) {
		position := protocol.Position{Line: line, Character: character}
		st := store.Store{Document: document, FileType: fileTypes[int(fileType)%len(fileTypes)]}

		items := completion.GetCompletions(testutils.NewTestLogger(t), st, position)

		if st.FileType == store.FileTypeValues || st.FileType == store.FileTypeUnknown {
			require.Empty(t, items)

			return
		}

		word := text.GetCursorWord(document, position)
		if word == "" {
			require.NotEmpty(t, items, "an empty cursor word matches the whole catalog")
		}

		for _, item := range items {
			require.True(t, strings.HasPrefix(item.Label, word), "label %q does not start with %q", item.Label, word)
		}
	})
}
