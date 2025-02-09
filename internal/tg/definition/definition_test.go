// // Package definition provides the logic for finding
// // definitions in Terragrunt configurations.
// package definition
//
// import (
// 	"bufio"
// 	"strings"
// 	"terragrunt-ls/internal/tg/store"
//
// 	"go.lsp.dev/protocol"
// 	"go.uber.org/zap"
// )
//
// const (
// 	// DefinitionContextInclude is the context for an include definition.
// 	// This means that the user is trying to find the definition of an include.
// 	DefinitionContextInclude = "include"
//
// 	// DefinitionContextNull is the context for a null definition.
// 	// This means that the user is trying to go to the definition of nothing useful.
// 	DefinitionContextNull = "null"
// )
//
// func GetDefinitionTargetWithContext(l *zap.SugaredLogger, store store.Store, position protocol.Position) (string, string) {
// 	document := store.Document
//
// 	scanner := bufio.NewScanner(strings.NewReader(document))
// 	target := ""
// 	definitionContext := ""
// 	lineHit := false
// 	scannedLines := 0
//
// 	for scanner.Scan() {
// 		line := scanner.Text()
//
// 		// Trim whitespace early to avoid
// 		// having to do it later.
// 		line = strings.TrimSpace(line)
//
// 		// Skip empty lines
// 		if len(line) == 0 {
// 			continue
// 		}
//
// 		// Skip comments
// 		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
// 			continue
// 		}
//
// 		// Identify configuration blocks
// 		block, labels, isBlock := isConfigBlockLine(line)
// 		if isBlock {
// 			l.Debugf("Found block: %s", block)
//
// 			if block == DefinitionContextInclude {
// 				definitionContext = DefinitionContextInclude
//
// 				// Includes can have zero labels
// 				if len(labels) > 0 {
// 					target = labels[0]
// 				}
// 			}
// 		}
//
// 		// Check if the current line is the one we're looking for
// 		if scannedLines == int(position.Line) {
// 			l.Debugf("Hit line %d: %s", position.Line, line)
//
// 			lineHit = true
// 		}
//
// 		// Check if we've reached the end of the block.
// 		//
// 		// End of blocks are special, as we've either discovered
// 		// the context for our definition, or we've reached
// 		// a point where we need to reset the context.
// 		//
// 		// The reason we do both checks is that we need to
// 		// account for single line block definitions.
// 		if line == "}" || (strings.HasSuffix(line, "}") && isBlock) {
// 			l.Debugf("End of block: %s, line: %d", block, scannedLines)
//
// 			if lineHit && definitionContext != "" {
// 				l.Debugf("Found target: %s, context: %s", target, definitionContext)
//
// 				return target, definitionContext
// 			}
//
// 			definitionContext = ""
// 			target = ""
// 		}
//
// 		scannedLines++
// 	}
//
// 	l.Debugf("No target found at %d:%d", position.Line, position.Character)
//
// 	return "", DefinitionContextNull
// }
//
// func isConfigBlockLine(line string) (string, []string, bool) {
// 	fields := strings.Fields(line)
//
// 	const minConfigBlockLen = 2
//
// 	if len(fields) < minConfigBlockLen {
// 		return "", nil, false
// 	}
//
// 	block := fields[0]
// 	labels := []string{}
//
// 	for _, field := range fields[1:] {
// 		if field == "=" {
// 			return "", nil, false
// 		}
//
// 		if field == "{" {
// 			return block, labels, true
// 		}
//
// 		labels = append(labels, strings.Trim(field, "\""))
// 	}
//
// 	return "", nil, false
// }

package definition_test

import (
	"terragrunt-ls/internal/testutils"
	"terragrunt-ls/internal/tg/definition"
	"terragrunt-ls/internal/tg/store"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
)

func TestGetDefinitionTargetWithContext(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name            string
		store           store.Store
		position        protocol.Position
		expectedTarget  string
		expectedContext string
	}{
		{
			name:            "empty store",
			store:           store.Store{},
			position:        protocol.Position{Line: 0, Character: 0},
			expectedTarget:  "",
			expectedContext: "null",
		},
		{
			name: "include definition",
			store: store.Store{
				Document: `include "root" {
	path = find_in_parent_folders("root")
}`,
			},
			position:        protocol.Position{Line: 0, Character: 0},
			expectedTarget:  "root",
			expectedContext: "include",
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l := testutils.NewTestLogger(t)

			target, context := definition.GetDefinitionTargetWithContext(l, tt.store, tt.position)

			assert.Equal(t, tt.expectedTarget, target)
			assert.Equal(t, tt.expectedContext, context)
		})
	}
}
