package lsp

import "go.lsp.dev/protocol"

type DidChangeTextDocumentNotification struct {
	Notification
	Params protocol.DidChangeTextDocumentParams `json:"params"`
}

// ContentChangeText returns the text of change, and false when change is a JSON null.
func ContentChangeText(change protocol.TextDocumentContentChangeEvent) (string, bool) {
	switch c := change.(type) {
	case *protocol.TextDocumentContentChangeWholeDocument:
		return c.Text, true
	case *protocol.TextDocumentContentChangePartial:
		return c.Text, true
	default:
		return "", false
	}
}
