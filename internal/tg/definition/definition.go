// Package definition provides the logic for finding
// definitions in Terragrunt configurations.
package definition

import (
	"bufio"
	"strings"
	"terragrunt-ls/internal/logger"
	"terragrunt-ls/internal/tg/store"

	"go.lsp.dev/protocol"
)

const (
	// DefinitionContextInclude is the context for an include definition.
	// This means that the user is trying to find the definition of an include.
	DefinitionContextInclude = "include"

	// DefinitionContextNull is the context for a null definition.
	// This means that the user is trying to go to the definition of nothing useful.
	DefinitionContextNull = "null"
)

func GetDefinitionTargetWithContext(l *logger.Logger, store store.Store, position protocol.Position) (string, string) {
	document := store.Document

	scanner := bufio.NewScanner(strings.NewReader(document))
	target := ""
	definitionContext := ""
	lineHit := false
	scannedLines := 0

	for scanner.Scan() {
		line := scanner.Text()

		// Trim whitespace early to avoid
		// having to do it later.
		line = strings.TrimSpace(line)

		// Skip empty lines
		if len(line) == 0 {
			continue
		}

		// Skip comments
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		// Identify configuration blocks
		block, labels, isBlock := isConfigBlockLine(line)
		if isBlock {
			l.Debug(
				"Found block",
				"block", block,
				"labels", labels,
				"line", scannedLines,
			)

			if block == DefinitionContextInclude {
				definitionContext = DefinitionContextInclude

				// Includes can have zero labels
				if len(labels) > 0 {
					target = labels[0]
				}
			}
		}

		// Check if the current line is the one we're looking for
		if scannedLines == int(position.Line) {
			l.Debug(
				"Hit line",
				"line", scannedLines,
				"position", position.Line,
			)

			lineHit = true
		}

		// Check if we've reached the end of the block.
		//
		// End of blocks are special, as we've either discovered
		// the context for our definition, or we've reached
		// a point where we need to reset the context.
		//
		// The reason we do both checks is that we need to
		// account for single line block definitions.
		if line == "}" || (strings.HasSuffix(line, "}") && isBlock) {
			l.Debug(
				"End of block",
				"block", block,
				"line", scannedLines,
			)

			if lineHit && definitionContext != "" {
				l.Debug(
					"Found target",
					"target", target,
					"context", definitionContext,
				)

				return target, definitionContext
			}

			definitionContext = ""
			target = ""
		}

		scannedLines++
	}

	l.Debug(
		"No target found",
		"line", position.Line,
		"character", position.Character,
	)

	return "", DefinitionContextNull
}

func isConfigBlockLine(line string) (string, []string, bool) {
	fields := strings.Fields(line)

	const minConfigBlockLen = 2

	if len(fields) < minConfigBlockLen {
		return "", nil, false
	}

	block := fields[0]
	labels := []string{}

	for _, field := range fields[1:] {
		if field == "=" {
			return "", nil, false
		}

		if field == "{" {
			return block, labels, true
		}

		labels = append(labels, strings.Trim(field, "\""))
	}

	return "", nil, false
}
