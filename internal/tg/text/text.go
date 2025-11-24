// Package text provides generic utilities for working with text.
package text

import (
	"bufio"
	"strings"

	"go.lsp.dev/protocol"
)

func GetCursorWord(document string, position protocol.Position) string {
	start, end := getWordBounds(document, position)
	return getLine(document, position.Line)[start:end]
}

// GetCursorWordRange returns the range of the word at the cursor position.
// Returns nil if no word is found at the position.
func GetCursorWordRange(document string, position protocol.Position) *protocol.Range {
	start, end := getWordBounds(document, position)

	// If no word found (start == end), return nil
	if start == end {
		return nil
	}

	return &protocol.Range{
		Start: protocol.Position{
			Line:      position.Line,
			Character: start,
		},
		End: protocol.Position{
			Line:      position.Line,
			Character: end,
		},
	}
}

// getLine returns the specified line from the document.
func getLine(document string, lineNum uint32) string {
	scanner := bufio.NewScanner(strings.NewReader(document))
	for i := 0; i <= int(lineNum); i++ {
		scanner.Scan()
	}
	return scanner.Text()
}

// getWordBounds finds the start and end character positions of a word at the given position.
func getWordBounds(document string, position protocol.Position) (start, end uint32) {
	line := getLine(document, position.Line)

	start = position.Character
	for start > 0 && int(start) <= len(line) && IsWordChar(line[start-1]) {
		start--
	}

	end = position.Character
	for int(end) < len(line) && IsWordChar(line[end]) {
		end++
	}

	return start, end
}

// IsWordChar checks if a character is part of a word (identifier).
func IsWordChar(c byte) bool {
	return c == '_' || c == '.' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

func WrapAsHCLCodeFence(s string) string {
	return "```hcl\n" + s + "\n```"
}
