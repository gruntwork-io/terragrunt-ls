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
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := text.GetCursorWord(tt.document, tt.position)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestGetCursorWordRange(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name     string
		document string
		expected *protocol.Range
		position protocol.Position
	}{
		{
			name:     "simple word at start",
			document: "hello",
			position: protocol.Position{Line: 0, Character: 0},
			expected: &protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 5},
			},
		},
		{
			name:     "simple word in middle",
			document: "hello",
			position: protocol.Position{Line: 0, Character: 2},
			expected: &protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 5},
			},
		},
		{
			name:     "local variable",
			document: "local.var",
			position: protocol.Position{Line: 0, Character: 6},
			expected: &protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 9},
			},
		},
		{
			name:     "second word",
			document: "hello world",
			position: protocol.Position{Line: 0, Character: 6},
			expected: &protocol.Range{
				Start: protocol.Position{Line: 0, Character: 6},
				End:   protocol.Position{Line: 0, Character: 11},
			},
		},
		{
			name:     "first word",
			document: "hello world",
			position: protocol.Position{Line: 0, Character: 2},
			expected: &protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 5},
			},
		},
		{
			name:     "whitespace only - no word",
			document: "   ",
			position: protocol.Position{Line: 0, Character: 1},
			expected: nil,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := text.GetCursorWordRange(tt.document, tt.position)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
