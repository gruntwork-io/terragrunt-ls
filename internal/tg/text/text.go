// Package text provides generic utilities for working with text.
package text

import (
	"bufio"
	"strings"

	"go.lsp.dev/protocol"
)

func GetCursorWord(document string, position protocol.Position) string {
	scanner := bufio.NewScanner(strings.NewReader(document))
	for i := 0; i <= int(position.Line); i++ {
		if !scanner.Scan() {
			return ""
		}
	}

	line := scanner.Text()

	// LSP treats a character past the end of the line as the end of the line.
	character := min(position.Character, uint32(len(line)))

	// Find the start of the word
	start := character
	for start > 0 && isWordChar(line[start-1]) {
		start--
	}

	// Find the end of the word
	end := character
	for int(end) < len(line) && isWordChar(line[end]) {
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
