package lsp

import "go.lsp.dev/protocol"

type DidOpenTextDocumentNotification struct {
	Notification
	Params protocol.DidOpenTextDocumentParams `json:"params"`
}
