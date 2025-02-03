package text

import (
	"bufio"
	"strings"

	"go.lsp.dev/protocol"
)

func GetCursorWord(document string, position protocol.Position) string {
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
