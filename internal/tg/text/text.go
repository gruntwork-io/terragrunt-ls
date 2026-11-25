// Package text provides generic utilities for working with text.
package text

import (
	"strings"
	"unicode/utf16"

	"go.lsp.dev/protocol"
)

// GetCursorWord returns the word under the cursor, or an empty string when the position is past the end of the
// document.
func GetCursorWord(document string, position protocol.Position) string {
	_, line, cursor := cursorLine(document, position)

	// Find the start of the word
	start := wordStart(line, cursor)

	// Find the end of the word
	end := cursor
	for end < len(line) && isWordChar(line[end]) {
		end++
	}

	return line[start:end]
}

// GetCursorWordPrefix returns the part of the word under the cursor that is before the cursor, and its range.
func GetCursorWordPrefix(document string, position protocol.Position) (string, protocol.Range) {
	_, line, cursor := cursorLine(document, position)
	prefix := line[wordStart(line, cursor):cursor]
	end := utf16Len(line[:cursor])

	return prefix, protocol.Range{
		Start: protocol.Position{Line: position.Line, Character: end - utf16Len(prefix)},
		End:   protocol.Position{Line: position.Line, Character: end},
	}
}

// GetCursorOffset returns the byte offset of the cursor in document.
func GetCursorOffset(document string, position protocol.Position) int {
	start, _, cursor := cursorLine(document, position)

	return start + cursor
}

func isWordChar(c byte) bool {
	return c == '_' || c == '.' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

func WrapAsHCLCodeFence(s string) string {
	return "```hcl\n" + s + "\n```"
}

// cursorLine returns the byte offset of the cursor line, its text, and the byte offset of the cursor in it.
func cursorLine(document string, position protocol.Position) (int, string, int) {
	start := 0

	for range position.Line {
		next := strings.IndexByte(document[start:], '\n')
		if next < 0 {
			return len(document), "", 0
		}

		start += next + 1
	}

	line := document[start:]
	if end := strings.IndexByte(line, '\n'); end >= 0 {
		line = line[:end]
	}

	line = strings.TrimSuffix(line, "\r")

	return start, line, byteOffset(line, position.Character)
}

// byteOffset returns the byte offset in line of a UTF-16 character offset, or len(line) past the end of line.
func byteOffset(line string, character uint32) int {
	var units uint32

	for i, r := range line {
		if units >= character {
			return i
		}

		units += runeUnits(r)
	}

	return len(line)
}

func utf16Len(s string) uint32 {
	var units uint32

	for _, r := range s {
		units += runeUnits(r)
	}

	return units
}

// surrogatePairUnits is the number of UTF-16 code units of a rune outside the Basic Multilingual Plane.
const surrogatePairUnits = 2

func runeUnits(r rune) uint32 {
	if utf16.RuneLen(r) == surrogatePairUnits {
		return surrogatePairUnits
	}

	return 1
}

func wordStart(line string, cursor int) int {
	start := cursor
	for start > 0 && isWordChar(line[start-1]) {
		start--
	}

	return start
}
