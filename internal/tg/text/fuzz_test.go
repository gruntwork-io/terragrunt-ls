package text_test

import (
	"regexp"
	"terragrunt-ls/internal/tg/text"
	"testing"

	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

var wordPattern = regexp.MustCompile(`^[A-Za-z0-9_.]*$`)

// FuzzGetCursorWord tests that the cursor word is always a run of word characters taken from the document.
func FuzzGetCursorWord(f *testing.F) {
	f.Add("locals {\n  foo = local.bar\n}", uint32(1), uint32(14))
	f.Add("h\u00e9llo w\u00f6rld", uint32(0), uint32(8))
	// Regression: a character past the end of the line panicked.
	f.Add("0", uint32(0), uint32(14))
	// Regression: a line past the end of the document panicked.
	f.Add("0", uint32(1), uint32(22))

	f.Fuzz(func(t *testing.T, document string, line, character uint32) {
		word := text.GetCursorWord(document, protocol.Position{Line: line, Character: character})

		require.Regexp(t, wordPattern, word)
		require.Contains(t, document, word)
	})
}
