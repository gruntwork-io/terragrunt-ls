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

// FuzzGetCompletions tests that each completion starts with the cursor word, or with the prefix for a variable field.
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
	f.Add("path.", uint32(0), uint32(5), uint8(0))
	f.Add("path.unit", uint32(0), uint32(5), uint8(1))

	f.Fuzz(func(t *testing.T, document string, line, character uint32, fileType uint8) {
		position := protocol.Position{Line: line, Character: character}
		st := &store.Store{Document: document, FileType: fileTypes[int(fileType)%len(fileTypes)]}

		items := completion.GetCompletions(testutils.NewTestLogger(t), testutils.NewDocs(t), st, position)

		if prefix, _ := text.GetCursorWordPrefix(document, position); strings.HasPrefix(prefix, "path.") {
			for i := range items {
				label := items[i].Label
				require.True(t, strings.HasPrefix(label, prefix), "label %q does not start with %q", label, prefix)
			}

			return
		}

		if st.FileType == store.FileTypeValues || st.FileType == store.FileTypeUnknown {
			require.Empty(t, items)

			return
		}

		word := text.GetCursorWord(document, position)
		if word == "" {
			require.NotEmpty(t, items, "an empty cursor word matches the whole catalog")
		}

		for i := range items {
			label := items[i].Label
			require.True(t, strings.HasPrefix(label, word), "label %q does not start with %q", label, word)
		}
	})
}
