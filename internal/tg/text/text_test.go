package text_test

import (
	"terragrunt-ls/internal/tg/text"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
)

func TestGetCursorWord(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name     string
		document string
		position protocol.Position
		expected string
	}{
		{
			name:     "simple word",
			document: "hello",
			position: protocol.Position{Line: 0, Character: 0},
			expected: "hello",
		},
		{
			name:     "local variable",
			document: "local.var",
			position: protocol.Position{Line: 0, Character: 0},
			expected: "local.var",
		},
		{
			name:     "two words",
			document: "hello world",
			position: protocol.Position{Line: 0, Character: 6},
			expected: "world",
		},
		{
			name:     "two words with cursor at the start",
			document: "hello world",
			position: protocol.Position{Line: 0, Character: 0},
			expected: "hello",
		},
		{
			name:     "two words with cursor in the middle",
			document: "hello world",
			position: protocol.Position{Line: 0, Character: 5},
			expected: "hello",
		},
		{
			name:     "two words with cursor at the end",
			document: "hello world",
			position: protocol.Position{Line: 0, Character: 11},
			expected: "world",
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := text.GetCursorWord(tt.document, tt.position)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
