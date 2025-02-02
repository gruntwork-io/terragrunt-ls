package hover

import (
	"bufio"
	"log"
	"strings"
	"terragrunt-ls/tg/store"

	"go.lsp.dev/protocol"
)

const (
	// HoverContextLocal is the context for a local hover.
	// This means that a hover is happening on top of a local variable.
	HoverContextLocal = "local"

	// HoverContextNull is the context for a null hover.
	// This means that a hover is happening on top of nothing useful.
	HoverContextNull = "null"
)

func GetHoverTargetWithContext(l *log.Logger, store store.Store, position protocol.Position) (string, string) {
	word := getHoveredWord(store.Document, position)
	if len(word) == 0 {
		l.Printf("No word found at %d:%d", position.Line, position.Character)

		return "", HoverContextNull
	}

	splitExpression := strings.Split(word, ".")
	if len(splitExpression) != 2 {
		l.Printf("Invalid word found at %d:%d: %s", position.Line, position.Character, word)

		return "", HoverContextNull
	}

	if splitExpression[0] == "local" {
		l.Printf("Found local variable: %s", splitExpression[1])

		return splitExpression[1], HoverContextLocal
	}

	return "", HoverContextNull
}

func getHoveredWord(document string, position protocol.Position) string {
	scanner := bufio.NewScanner(strings.NewReader(document))
	for i := 0; i <= int(position.Line); i++ {
		scanner.Scan()
	}

	line := scanner.Text()

	// Find the start of the word
	start := position.Character
	for start > 0 && isWordChar(line[start-1]) {
		start--
	}

	// Find the end of the word
	end := position.Character
	for int(end) < len(line) && isWordChar(line[end]) {
		end++
	}

	return line[start:end]
}

func isWordChar(c byte) bool {
	return c == '_' || c == '.' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}
