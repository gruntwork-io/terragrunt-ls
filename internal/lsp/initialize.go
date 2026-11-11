package lsp

import "go.lsp.dev/protocol"

type InitializeRequest struct {
	Params protocol.InitializeParams `json:"params"`
	Request
}

type InitializeResponse struct {
	Response
	Result protocol.InitializeResult `json:"result"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func NewInitializeResponse(id int) InitializeResponse {
	return InitializeResponse{
		RPC: RPCVersion,
		ID:  &id,
		Result: protocol.InitializeResult{
			Capabilities: protocol.ServerCapabilities{
				TextDocumentSync:           protocol.TextDocumentSyncKindFull,
				HoverProvider:              protocol.Boolean(true),
				DefinitionProvider:         protocol.Boolean(true),
				ReferencesProvider:         protocol.Boolean(true),
				CompletionProvider:         &protocol.CompletionOptions{},
				DocumentFormattingProvider: protocol.Boolean(true),
				RenameProvider: &protocol.RenameOptions{
					PrepareProvider: new(true),
				},
			},
			ServerInfo: protocol.ServerInfo{
				Name:    name,
				Version: protocol.NewOptional(version),
			},
		},
	}
}
