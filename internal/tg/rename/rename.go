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

		// Check if it's a standalone identifier (not part of a larger word)
		isStandalone := true

		// Check if it's inside a string (skip if it is)
		if isInsideString(line, absolutePos) {
			isStandalone = false
		}

		// Check character before
		if absolutePos > 0 && isStandalone {
			prevChar := line[absolutePos-1]
			if isWordChar(prevChar) {
				isStandalone = false
			}
		}

		// Check character after
		endPos := absolutePos + len(identifier)
		if endPos < len(line) && isStandalone {
			nextChar := line[endPos]
			if isWordChar(nextChar) {
				isStandalone = false
			}
		}

		// Also skip if it's part of "local.identifier" (we handle those separately)
		if absolutePos >= 6 && isStandalone && line[absolutePos-6:absolutePos] == "local." {
			isStandalone = false
		}

		if isStandalone {
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

		startPos = absolutePos + len(identifier)
	}

	return ranges
}

// isInsideString checks if a position in a line is inside a string literal.
// Handles both double quotes (") and heredocs (<<EOF, <<-EOF).
func isInsideString(line string, pos int) bool {
	inDoubleQuote := false
	inSingleQuote := false

	// Check for heredoc syntax - if line contains <<, it's likely a heredoc start
	// For simplicity, we'll treat the whole line as non-string in heredoc cases
	// A more robust solution would track heredoc state across lines
	if strings.Contains(line, "<<") {
		// Don't treat identifier definitions in heredoc lines as strings
		// This is a simplified approach
		return false
	}

	// Walk through the line up to the position
	for i := 0; i < pos && i < len(line); i++ {
		char := line[i]

		// Handle escape sequences
		if char == '\\' && i+1 < len(line) {
			i++ // Skip next character
			continue
		}

		// Toggle double quote state
		if char == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
		}

		// Toggle single quote state (for HCL)
		if char == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
		}
	}

	return inDoubleQuote || inSingleQuote
}

// isWordChar checks if a character is part of a word (identifier).
func isWordChar(c byte) bool {
	return c == '_' || c == '.' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}
