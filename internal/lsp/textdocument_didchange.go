package lsp

import "go.lsp.dev/protocol"

type DidChangeTextDocumentNotification struct {
	Notification
	Params protocol.DidChangeTextDocumentParams `json:"params"`
}
