package text_test

import (
	"terragrunt-ls/internal/tg/text"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
)

// TestGetCursorWord tests the word found at a position, including positions past the end of a line or the document.
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
		{
			name:     "after characters of two and four bytes",
			document: "\u00e9\U0001F600 local.x",
			position: protocol.Position{Line: 0, Character: 5},
			expected: "local.x",
		},
		{
			name:     "line that ends with CRLF",
			document: "a = local.x\r\nb",
			position: protocol.Position{Line: 0, Character: 40},
			expected: "local.x",
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

func TestGetCursorWordPrefix(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name     string
		document string
		expected string
		position protocol.Position
		start    uint32
		end      uint32
	}{
		{
			name:     "after a dot",
			document: "  x = path.",
			position: protocol.Position{Character: 11},
			expected: "path.",
			start:    6,
			end:      11,
		},
		{
			name:     "in a word",
			document: "path.unit",
			position: protocol.Position{Character: 6},
			expected: "path.u",
			start:    0,
			end:      6,
		},
		{
			name:     "at the start of a word",
			document: "path",
			position: protocol.Position{Character: 0},
			expected: "",
			start:    0,
			end:      0,
		},
		{
			name:     "after a space",
			document: "path ",
			position: protocol.Position{Character: 5},
			expected: "",
			start:    5,
			end:      5,
		},
		{
			name:     "past the end of the line",
			document: "a = path.u\nb",
			position: protocol.Position{Character: 40},
			expected: "path.u",
			start:    4,
			end:      10,
		},
		{
			name:     "on a later line",
			document: "a\n  path.s",
			position: protocol.Position{Line: 1, Character: 8},
			expected: "path.s",
			start:    2,
			end:      8,
		},
		{
			name:     "past the end of the document",
			document: "path.",
			position: protocol.Position{Line: 3, Character: 1},
			expected: "",
			start:    0,
			end:      0,
		},
		{
			name:     "after characters of two and four bytes",
			document: "  x = \"\u00e9\U0001F600${path.u",
			position: protocol.Position{Character: 18},
			expected: "path.u",
			start:    12,
			end:      18,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			prefix, rng := text.GetCursorWordPrefix(tt.document, tt.position)
			assert.Equal(t, tt.expected, prefix)
			assert.Equal(t, protocol.Range{
				Start: protocol.Position{Line: tt.position.Line, Character: tt.start},
				End:   protocol.Position{Line: tt.position.Line, Character: tt.end},
			}, rng)
		})
	}
}

func TestGetCursorOffset(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name     string
		document string
		position protocol.Position
		expected int
	}{
		{name: "first line", document: "a = 1\nb = 2", position: protocol.Position{Character: 3}, expected: 3},
		{name: "later line", document: "a = 1\nb = 2", position: protocol.Position{Line: 1, Character: 1}, expected: 7},
		{name: "past the end of a line", document: "a\r\nb", position: protocol.Position{Character: 9}, expected: 1},
		{name: "past the end of the document", document: "a\nb", position: protocol.Position{Line: 7}, expected: 3},
		{
			name:     "after a character of four bytes",
			document: "\U0001F600x",
			position: protocol.Position{Character: 2},
			expected: 4,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, text.GetCursorOffset(tt.document, tt.position))
		})
	}
}
