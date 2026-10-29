package hover_test

import (
	"terragrunt-ls/internal/testutils"
	"terragrunt-ls/internal/tg/hover"
	"terragrunt-ls/internal/tg/store"
	"terragrunt-ls/internal/tg/text"
	"testing"

	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

// FuzzGetHoverTargetWithContext tests that hover only reports a local for a `local.<name>` cursor word.
func FuzzGetHoverTargetWithContext(f *testing.F) {
	f.Add("locals {\n  foo = local.bar\n}", uint32(1), uint32(14))
	f.Add("locals {\n  foo = local.bar.baz\n}", uint32(1), uint32(14))
	// Regression: a line past the end of the document panicked.
	f.Add("0", uint32(1), uint32(22))

	f.Fuzz(func(t *testing.T, document string, line, character uint32) {
		position := protocol.Position{Line: line, Character: character}
		st := store.Store{Document: document, FileType: store.FileTypeUnit}

		target, hoverContext := hover.GetHoverTargetWithContext(testutils.NewTestLogger(t), st, position)

		word := text.GetCursorWord(document, position)
		if hoverContext == hover.HoverContextLocal {
			require.Equal(t, "local."+target, word)

			return
		}

		require.Equal(t, hover.HoverContextNull, hoverContext)
		require.Equal(t, word, target)
	})
}
