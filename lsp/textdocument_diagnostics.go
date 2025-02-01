package lsp

import "go.lsp.dev/protocol"

type PublishDiagnosticsNotification struct {
	Notification
	Params protocol.PublishDiagnosticsParams `json:"params"`
}
