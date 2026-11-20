// Package text provides generic utilities for working with text.
package text

import (
	"bufio"
	"strings"
	"terragrunt-ls/internal/ast"

	"go.lsp.dev/protocol"
)

// GetCursorWord returns the word under the cursor, or an empty string when the position is past the end of the
// document.
func GetCursorWord(document string, position protocol.Position) string {
	scanner := bufio.NewScanner(strings.NewReader(document))
	for i := 0; i <= int(position.Line); i++ {
		if !scanner.Scan() {
			return ""
		}
	}

	line := scanner.Text()
	cursor := ast.LineByteOffset(line, position.Character)

	// Find the start of the word
	start := cursor
	for start > 0 && isWordChar(line[start-1]) {
		start--
	}

	// Find the end of the word
	end := cursor
	for end < len(line) && isWordChar(line[end]) {
		end++
	}

	return line[start:end]
}

func isWordChar(c byte) bool {
	return c == '_' || c == '.' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

func WrapAsHCLCodeFence(s string) string {
	return "```hcl\n" + s + "\n```"
}
