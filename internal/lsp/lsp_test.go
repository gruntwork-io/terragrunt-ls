package lsp_test

import (
	"terragrunt-ls/internal/lsp"
	"terragrunt-ls/internal/rpc"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

func TestInitializeResponseEncoding(t *testing.T) {
	t.Parallel()

	expected := `{
		"id": 1,
		"jsonrpc": "2.0",
		"result": {
			"capabilities": {
				"completionProvider": {},
				"definitionProvider": true,
				"documentFormattingProvider": true,
				"hoverProvider": true,
				"referencesProvider": true,
				"renameProvider": {"prepareProvider": true},
				"textDocumentSync": 1
			},
			"serverInfo": {"name": "terragrunt-ls", "version": "0.1.0"}
		}
	}`

	assert.JSONEq(t, expected, encodedBody(t, lsp.NewInitializeResponse(1)))
}

func TestPublishDiagnosticsEncoding(t *testing.T) {
	t.Parallel()

	msg := lsp.PublishDiagnosticsNotification{
		RPC:    lsp.RPCVersion,
		Method: protocol.MethodTextDocumentPublishDiagnostics,
		Params: protocol.PublishDiagnosticsParams{
			URI: "file:///unit/terragrunt.hcl",
			Diagnostics: []protocol.Diagnostic{{
				Severity: protocol.DiagnosticSeverityError,
				Source:   protocol.NewOptional("HCL"),
				Message:  protocol.String("Invalid expression"),
			}},
		},
	}

	expected := `{
		"jsonrpc": "2.0",
		"method": "textDocument/publishDiagnostics",
		"params": {
			"uri": "file:///unit/terragrunt.hcl",
			"diagnostics": [{
				"message": "Invalid expression",
				"range": {"start": {"line": 0, "character": 0}, "end": {"line": 0, "character": 0}},
				"severity": 1,
				"source": "HCL"
			}]
		}
	}`

	assert.JSONEq(t, expected, encodedBody(t, msg))
}

func TestCompletionResponseEncoding(t *testing.T) {
	t.Parallel()

	msg := lsp.CompletionResponse{
		RPC: lsp.RPCVersion, ID: new(7),
		Result: []protocol.CompletionItem{{
			Label: "locals",
			Documentation: &protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: "# locals",
			},
			Kind:             protocol.CompletionItemKindClass,
			InsertTextFormat: protocol.InsertTextFormatSnippet,
			TextEdit:         &protocol.TextEdit{NewText: "locals {}"},
		}},
	}

	expected := `{
		"id": 7,
		"jsonrpc": "2.0",
		"result": [{
			"documentation": {"kind": "markdown", "value": "# locals"},
			"insertTextFormat": 2,
			"kind": 7,
			"label": "locals",
			"textEdit": {
				"newText": "locals {}",
				"range": {"start": {"line": 0, "character": 0}, "end": {"line": 0, "character": 0}}
			}
		}]
	}`

	assert.JSONEq(t, expected, encodedBody(t, msg))
}

func TestHoverResponseEncoding(t *testing.T) {
	t.Parallel()

	tc := []struct {
		result   *lsp.HoverResult
		name     string
		expected string
	}{
		{
			name: "markdown",
			result: &lsp.HoverResult{
				Contents: protocol.MarkupContent{
					Kind:  protocol.MarkupKindMarkdown,
					Value: "# locals",
				},
			},
			expected: `{
				"id": 7,
				"jsonrpc": "2.0",
				"result": {"contents": {"kind": "markdown", "value": "# locals"}}
			}`,
		},
		{
			name:     "nothing to show",
			result:   nil,
			expected: `{"id": 7, "jsonrpc": "2.0", "result": null}`,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			msg := lsp.HoverResponse{
				Response: lsp.Response{RPC: lsp.RPCVersion, ID: new(7)},
				Result:   tt.result,
			}

			assert.JSONEq(t, tt.expected, encodedBody(t, msg))
		})
	}
}

func TestInitializeRequestDecoding(t *testing.T) {
	t.Parallel()

	content := []byte(`{
		"jsonrpc": "2.0",
		"id": 3,
		"method": "initialize",
		"params": {
			"processId": null,
			"rootUri": null,
			"capabilities": {},
			"clientInfo": {"name": "editor", "version": "1.2.3"}
		}
	}`)

	var request lsp.InitializeRequest

	require.NoError(t, protocol.Unmarshal(content, &request))

	assert.Equal(t, 3, request.ID)
	assert.Equal(t, protocol.MethodInitialize, request.Method)
	assert.Equal(t, "editor", request.Params.ClientInfo.Name)
}

func TestReferencesRequestDecoding(t *testing.T) {
	t.Parallel()

	content := []byte(`{
		"jsonrpc": "2.0",
		"id": 4,
		"method": "textDocument/references",
		"params": {
			"textDocument": {"uri": "file:///my%20unit/terragrunt.hcl"},
			"position": {"line": 2, "character": 5},
			"context": {"includeDeclaration": true}
		}
	}`)

	var request lsp.ReferencesRequest

	require.NoError(t, protocol.Unmarshal(content, &request))

	assert.Equal(t, 4, request.ID)
	assert.Equal(t, protocol.Position{Line: 2, Character: 5}, request.Params.Position)
	assert.True(t, request.Params.Context.IncludeDeclaration)
	assert.Equal(t, "/my unit/terragrunt.hcl", request.Params.TextDocument.URI.Path())
}

func TestContentChangeText(t *testing.T) {
	t.Parallel()

	tc := []struct {
		name         string
		change       string
		expectedText string
		expectedOK   bool
	}{
		{
			name:         "whole document",
			change:       `{"text": "locals {}"}`,
			expectedText: "locals {}",
			expectedOK:   true,
		},
		{
			name:         "partial document",
			change:       `{"range": {"start": {"line": 0, "character": 0}, "end": {"line": 0, "character": 1}}, "text": "x"}`,
			expectedText: "x",
			expectedOK:   true,
		},
		{
			name:         "null",
			change:       `null`,
			expectedText: "",
			expectedOK:   false,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			content := []byte(`{
				"jsonrpc": "2.0",
				"method": "textDocument/didChange",
				"params": {
					"textDocument": {"uri": "file:///terragrunt.hcl", "version": 2},
					"contentChanges": [` + tt.change + `]
				}
			}`)

			var notification lsp.DidChangeTextDocumentNotification

			require.NoError(t, protocol.Unmarshal(content, &notification))
			require.Len(t, notification.Params.ContentChanges, 1)

			text, ok := lsp.ContentChangeText(notification.Params.ContentChanges[0])

			assert.Equal(t, tt.expectedOK, ok)
			assert.Equal(t, tt.expectedText, text)
		})
	}
}

// encodedBody returns the JSON body of msg as written to the client.
func encodedBody(t *testing.T, msg any) string {
	t.Helper()

	_, content, err := rpc.DecodeMessage([]byte(rpc.EncodeMessage(msg)))
	require.NoError(t, err)

	return string(content)
}
