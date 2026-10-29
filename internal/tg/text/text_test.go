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
		expected string
		position protocol.Position
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
		{
			name:     "cursor past the end of the line",
			document: "hello world",
			position: protocol.Position{Line: 0, Character: 40},
			expected: "world",
		},
		{
			name:     "cursor past the end of a shorter line",
			document: "hello\nhi",
			position: protocol.Position{Line: 1, Character: 4},
			expected: "hi",
		},
		{
			name:     "line past the end of the document",
			document: "hello\nworld",
			position: protocol.Position{Line: 5, Character: 2},
			expected: "",
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
