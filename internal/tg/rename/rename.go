// Package rename provides the logic for renaming identifiers in Terragrunt configurations.
package rename

import (
	"bufio"
	"strings"
	"terragrunt-ls/internal/logger"
	"terragrunt-ls/internal/tg/store"
	"terragrunt-ls/internal/tg/text"

	"go.lsp.dev/protocol"
)

const (
	// RenameContextLocal indicates a local variable is being renamed.
	RenameContextLocal = "local"

	// RenameContextNull indicates the position is not renameable.
	RenameContextNull = "null"
)

// GetRenameTargetWithContext determines what is being renamed at the given position.
// Returns the target name and the context type.
func GetRenameTargetWithContext(l logger.Logger, store store.Store, position protocol.Position) (string, string) {
	word := text.GetCursorWord(store.Document, position)
	if len(word) == 0 {
		l.Debug(
			"No word found for rename",
			"line", position.Line,
			"character", position.Character,
		)
		return "", RenameContextNull
	}

	splitExpression := strings.Split(word, ".")

	const localPartsLen = 2

	// Check if it's a local variable reference (local.varname)
	if len(splitExpression) == localPartsLen && splitExpression[0] == "local" {
		l.Debug(
			"Found local variable for rename",
			"line", position.Line,
			"character", position.Character,
			"local", splitExpression[1],
		)
		return splitExpression[1], RenameContextLocal
	}

	l.Debug(
		"No renameable identifier found",
		"line", position.Line,
		"character", position.Character,
		"word", word,
	)

	return "", RenameContextNull
}

// FindAllOccurrences finds all occurrences of an identifier in the document.
// For local variables, it finds both the definition and all references (local.varname).
func FindAllOccurrences(l logger.Logger, document string, identifier string, context string) []protocol.Range {
	var ranges []protocol.Range

	scanner := bufio.NewScanner(strings.NewReader(document))
	lineNum := uint32(0)

	for scanner.Scan() {
		line := scanner.Text()

		switch context {
		case RenameContextLocal:
			// Find occurrences of "local.identifier"
			fullIdentifier := "local." + identifier
			ranges = append(ranges, findInLine(line, fullIdentifier, lineNum, 6)...) // 6 = len("local.")

			// Also find the definition in locals block (just "identifier")
			// Only match standalone identifiers (not part of longer words)
			ranges = append(ranges, findStandaloneIdentifierInLine(line, identifier, lineNum)...)
		}

		lineNum++
	}

	l.Debug(
		"Found occurrences",
		"identifier", identifier,
		"count", len(ranges),
		"context", context,
	)

	return ranges
}

// findInLine finds all occurrences of a string in a line and returns ranges.
// offset indicates how many characters from the start of the match should be the actual rename position.
func findInLine(line string, search string, lineNum uint32, offset int) []protocol.Range {
	var ranges []protocol.Range
	startPos := 0

	for {
		idx := strings.Index(line[startPos:], search)
		if idx == -1 {
			break
		}

		absolutePos := startPos + idx
		ranges = append(ranges, protocol.Range{
			Start: protocol.Position{
				Line:      lineNum,
				Character: uint32(absolutePos + offset),
			},
			End: protocol.Position{
				Line:      lineNum,
				Character: uint32(absolutePos + len(search)),
			},
		})

		startPos = absolutePos + len(search)
	}

	return ranges
}

// findStandaloneIdentifierInLine finds standalone occurrences of an identifier
// (not part of a larger word) in a line.
func findStandaloneIdentifierInLine(line string, identifier string, lineNum uint32) []protocol.Range {
	var ranges []protocol.Range
	startPos := 0

	for {
		idx := strings.Index(line[startPos:], identifier)
		if idx == -1 {
			break
		}

		absolutePos := startPos + idx
		endPos := absolutePos + len(identifier)

		if isStandaloneIdentifier(line, absolutePos, endPos) {
			ranges = append(ranges, protocol.Range{
				Start: protocol.Position{
					Line:      lineNum,
					Character: uint32(absolutePos),
				},
				End: protocol.Position{
					Line:      lineNum,
					Character: uint32(endPos),
				},
			})
		}

		startPos = endPos
	}

	return ranges
}

// isStandaloneIdentifier checks if an identifier at the given position is standalone
// (not inside a string, not part of a larger word, not part of local.identifier).
func isStandaloneIdentifier(line string, start, end int) bool {
	// Check if it's inside a string
	if isInsideString(line, start) {
		return false
	}

	// Check character before
	if start > 0 && text.IsWordChar(line[start-1]) {
		return false
	}

	// Check character after
	if end < len(line) && text.IsWordChar(line[end]) {
		return false
	}

	// Skip if it's part of "local.identifier" (we handle those separately)
	if start >= 6 && line[start-6:start] == "local." {
		return false
	}

	return true
}

// isInsideString checks if a position in a line is inside a string literal.
func isInsideString(line string, pos int) bool {
	// Heredoc lines are not treated as strings for identifier matching
	if strings.Contains(line, "<<") {
		return false
	}

	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < pos && i < len(line); i++ {
		char := line[i]

		// Handle escape sequences
		if char == '\\' && i+1 < len(line) {
			i++ // Skip next character
			continue
		}

		// Toggle quote state
		if char == '"' || char == '\'' {
			if !inQuote {
				inQuote = true
				quoteChar = char
			} else if char == quoteChar {
				inQuote = false
				quoteChar = 0
			}
		}
	}

	return inQuote
}
