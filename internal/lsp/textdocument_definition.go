package lsp

import "go.lsp.dev/protocol"

type DefinitionRequest struct {
	Params protocol.DefinitionParams `json:"params"`
	Request
}

type DefinitionResponse struct {
	Response
	Result protocol.Location `json:"result"`
}
