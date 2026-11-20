package ast_test

import (
	"strings"
	"terragrunt-ls/internal/ast"
	"testing"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func TestByteOffset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		doc      string
		position protocol.Position
		expected int
	}{
		{
			name:     "ascii",
			doc:      `a = "x"`,
			position: protocol.Position{Line: 0, Character: 5},
			expected: 5,
		},
		{
			name:     "after surrogate pairs",
			doc:      `a = "🧪🧪x"`,
			position: protocol.Position{Line: 0, Character: 9},
			expected: 13,
		},
		{
			name:     "inside a surrogate pair",
			doc:      `a = "🧪x"`,
			position: protocol.Position{Line: 0, Character: 6},
			expected: 5,
		},
		{
			name:     "after a combining accent",
			doc:      "a = \"e\u0301x\"",
			position: protocol.Position{Line: 0, Character: 7},
			expected: 8,
		},
		{
			name:     "after cjk",
			doc:      "a = \"\u65e5x\"",
			position: protocol.Position{Line: 0, Character: 6},
			expected: 8,
		},
		{
			name:     "second line",
			doc:      "a = 1\nb = \"🧪x\"",
			position: protocol.Position{Line: 1, Character: 7},
			expected: 15,
		},
		{
			name:     "crlf line ending",
			doc:      "a = 1\r\nb = 2",
			position: protocol.Position{Line: 1, Character: 0},
			expected: 7,
		},
		{
			name:     "past the end of a crlf line",
			doc:      "a = 1\r\nb = 2",
			position: protocol.Position{Line: 0, Character: 40},
			expected: 5,
		},
		{
			name:     "past the end of the line",
			doc:      "a = \"🧪\"\nb = 2",
			position: protocol.Position{Line: 0, Character: 40},
			expected: 10,
		},
		{
			name:     "past the end of the document",
			doc:      "a = 1",
			position: protocol.Position{Line: 3, Character: 2},
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, ast.ByteOffset(tt.doc, tt.position))
		})
	}
}

func TestFromHCLPos(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		doc      string
		pos      hcl.Pos
		expected protocol.Position
	}{
		{
			name:     "ascii",
			doc:      `a = "x"`,
			pos:      hcl.Pos{Line: 1, Column: 6, Byte: 5},
			expected: protocol.Position{Line: 0, Character: 5},
		},
		{
			name:     "after surrogate pairs",
			doc:      `a = "🧪🧪x"`,
			pos:      hcl.Pos{Line: 1, Column: 8, Byte: 13},
			expected: protocol.Position{Line: 0, Character: 9},
		},
		{
			name:     "after a combining accent",
			doc:      "a = \"e\u0301x\"",
			pos:      hcl.Pos{Line: 1, Column: 7, Byte: 8},
			expected: protocol.Position{Line: 0, Character: 7},
		},
		{
			name:     "after cjk",
			doc:      "a = \"\u65e5x\"",
			pos:      hcl.Pos{Line: 1, Column: 7, Byte: 8},
			expected: protocol.Position{Line: 0, Character: 6},
		},
		{
			name:     "end of the document",
			doc:      "a = \"🧪\"",
			pos:      hcl.Pos{Line: 1, Column: 8, Byte: 10},
			expected: protocol.Position{Line: 0, Character: 8},
		},
		{
			name:     "zero position",
			doc:      "a = 1",
			pos:      hcl.Pos{},
			expected: protocol.Position{Line: 0, Character: 0},
		},
		{
			name:     "line past the end of the document",
			doc:      "a = 1",
			pos:      hcl.Pos{Line: 4, Column: 3},
			expected: protocol.Position{Line: 3, Character: 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, ast.FromHCLPos(tt.doc, tt.pos))
		})
	}
}

func TestRebaseHCLPos(t *testing.T) {
	t.Parallel()

	const (
		from = "include \"\" {\n}\na = local.x.z"
		to   = "include {\n}\na = local.x.z"
	)

	tests := []struct {
		name     string
		pos      hcl.Pos
		expected hcl.Pos
	}{
		{
			name:     "unedited line",
			pos:      hcl.Pos{Line: 3, Column: 12, Byte: 26},
			expected: hcl.Pos{Line: 3, Column: 12, Byte: 23},
		},
		{
			name:     "end of the document",
			pos:      hcl.Pos{Line: 3, Column: 14, Byte: 28},
			expected: hcl.Pos{Line: 3, Column: 14, Byte: 25},
		},
		{
			name:     "past the end of its line in to",
			pos:      hcl.Pos{Line: 1, Column: 12, Byte: 11},
			expected: hcl.Pos{Line: 1, Column: 12, Byte: 9},
		},
		{
			name:     "zero position",
			pos:      hcl.Pos{},
			expected: hcl.Pos{},
		},
		{
			name:     "line past the end of both documents",
			pos:      hcl.Pos{Line: 9, Column: 1, Byte: 28},
			expected: hcl.Pos{Line: 9, Column: 1, Byte: 25},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, ast.RebaseHCLPos(from, to, tt.pos))
		})
	}
}

// FuzzPositionsMatchHCL pins that the LSP position of the start of every token HCL lexes is on
// the token's line, converts back to the token's byte offset, and counts the UTF-16 code units
// before the token on its line.
func FuzzPositionsMatchHCL(f *testing.F) {
	f.Add("locals {\n  env = \"dev\"\n}\n")
	f.Add("a = \"🧪🧪x\"\nb = local.a\n")
	f.Add("a = \"e\u0301\u0301x\" # \u65e5\u672c\r\nb = 1")
	f.Add("a = <<EOT\n🧪 ${local.x}\nEOT\n")

	f.Fuzz(func(t *testing.T, doc string) {
		if !utf8.ValidString(doc) {
			t.Skip("documents arrive as JSON strings, which are valid UTF-8")
		}

		tokens, _ := hclsyntax.LexConfig([]byte(doc), "terragrunt.hcl", hcl.InitialPos)

		for _, token := range tokens {
			start := token.Range.Start

			// LSP treats "\r\n" as one line ending, so no position falls between its bytes.
			if strings.HasSuffix(doc[:start.Byte], "\r") && strings.HasPrefix(doc[start.Byte:], "\n") {
				continue
			}

			lspPos := ast.FromHCLPos(doc, start)
			require.Equal(t, uint32(start.Line-1), lspPos.Line, "token %q", token.Bytes)
			require.Equal(t, start.Byte, ast.ByteOffset(doc, lspPos), "token %q", token.Bytes)

			lineStart := start.Byte
			for lineStart > 0 && doc[lineStart-1] != '\n' {
				lineStart--
			}

			require.Equal(t, uint32(len(utf16.Encode([]rune(doc[lineStart:start.Byte])))), lspPos.Character)
		}
	})
}
