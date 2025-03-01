package main

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"terragrunt-ls/internal/logger"
	"terragrunt-ls/internal/lsp"
	"terragrunt-ls/internal/rpc"
	"terragrunt-ls/internal/tg"

	"go.lsp.dev/protocol"
)

func main() {
	logfile := os.Getenv("TG_LS_LOG")

	l := logger.NewLogger(logfile)
	defer func() {
		if err := l.Close(); err != nil {
			panic(err)
		}
	}()

	l.Info("Initializing terragrunt-ls")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(rpc.Split)

	state := tg.NewState()
	writer := os.Stdout

	for scanner.Scan() {
		msg := scanner.Bytes()

		method, contents, err := rpc.DecodeMessage(msg)
		if err != nil {
			l.Error("Got an error decoding message from client", "err", err)

			continue
		}

		handleMessage(l, writer, state, method, contents)
	}
}

func handleMessage(l *logger.Logger, writer io.Writer, state tg.State, method string, contents []byte) {
	l.Debug("Received msg", "method", method, "contents", string(contents))

	switch method {
	case protocol.MethodInitialize:
		var request lsp.InitializeRequest
		if err := json.Unmarshal(contents, &request); err != nil {
			l.Error("Failed to parse initialize request", "err", err)
		}

		l.Debug("Connected",
			"Name", request.Params.ClientInfo.Name,
			"Version", request.Params.ClientInfo.Version)

		msg := lsp.NewInitializeResponse(request.ID)
		writeResponse(l, writer, msg)

		l.Debug("Initialized")

	case protocol.MethodTextDocumentDidOpen:
		var notification lsp.DidOpenTextDocumentNotification
		if err := json.Unmarshal(contents, &notification); err != nil {
			l.Error(
				"Failed to parse didOpen request",
				"error",
				err,
			)
		}

		l.Debug(
			"Opened",
			"URI", notification.Params.TextDocument.URI,
			"LanguageID", notification.Params.TextDocument.LanguageID,
			"Version", notification.Params.TextDocument.Version,
			"Text", notification.Params.TextDocument.Text,
		)

		diagnostics := state.OpenDocument(l, notification.Params.TextDocument.URI, notification.Params.TextDocument.Text)
		writeResponse(l, writer, lsp.PublishDiagnosticsNotification{
			Notification: lsp.Notification{
				RPC:    lsp.RPCVersion,
				Method: protocol.MethodTextDocumentPublishDiagnostics,
			},
			Params: protocol.PublishDiagnosticsParams{
				URI:         notification.Params.TextDocument.URI,
				Diagnostics: diagnostics,
			},
		})

		l.Debug(
			"Document opened",
			"URI", notification.Params.TextDocument.URI,
		)

	case protocol.MethodTextDocumentDidChange:
		var notification lsp.DidChangeTextDocumentNotification
		if err := json.Unmarshal(contents, &notification); err != nil {
			l.Error(
				"Failed to parse didChange request",
				"error",
				err,
			)
		}

		l.Debug(
			"Changed",
			"URI", notification.Params.TextDocument.URI,
			"Changes", notification.Params.ContentChanges,
		)

		for _, change := range notification.Params.ContentChanges {
			l.Debug(
				"Change",
				"Range", change.Range,
				"Text", change.Text,
			)

			diagnostics := state.UpdateDocument(l, notification.Params.TextDocument.URI, change.Text)
			writeResponse(l, writer, lsp.PublishDiagnosticsNotification{
				Notification: lsp.Notification{
					RPC:    lsp.RPCVersion,
					Method: protocol.MethodTextDocumentPublishDiagnostics,
				},
				Params: protocol.PublishDiagnosticsParams{
					URI:         notification.Params.TextDocument.URI,
					Diagnostics: diagnostics,
				},
			})
		}

		l.Debug(
			"Document changed",
			"URI", notification.Params.TextDocument.URI,
		)

	case protocol.MethodTextDocumentHover:
		var request lsp.HoverRequest
		if err := json.Unmarshal(contents, &request); err != nil {
			l.Debug(
				"Failed to parse hover request",
				"error",
				err,
			)
		}

		l.Debug(
			"Hover",
			"URI", request.Params.TextDocument.URI,
			"Position", request.Params.Position,
		)

		response := state.Hover(l, request.ID, request.Params.TextDocument.URI, request.Params.Position)

		writeResponse(l, writer, response)

	case protocol.MethodTextDocumentDefinition:
		var request lsp.DefinitionRequest
		if err := json.Unmarshal(contents, &request); err != nil {
			l.Error(
				"Failed to parse definition request",
				"error",
				err,
			)
		}

		l.Debug(
			"Definition",
			"URI", request.Params.TextDocument.URI,
			"Position", request.Params.Position,
		)

		response := state.Definition(l, request.ID, request.Params.TextDocument.URI, request.Params.Position)

		writeResponse(l, writer, response)

	case protocol.MethodTextDocumentCompletion:
		var request lsp.CompletionRequest
		if err := json.Unmarshal(contents, &request); err != nil {
			l.Error(
				"Failed to parse completion request",
				"error",
				err,
			)
		}

		l.Debug(
			"Completion",
			"URI", request.Params.TextDocument.URI,
			"Position", request.Params.Position,
		)

		response := state.TextDocumentCompletion(l, request.ID, request.Params.TextDocument.URI, request.Params.Position)

		l.Debug(
			"Completion response",
			"Response", response,
		)

		writeResponse(l, writer, response)
	}
}

func writeResponse(l *logger.Logger, writer io.Writer, msg any) {
	reply := rpc.EncodeMessage(msg)

	_, err := writer.Write([]byte(reply))
	if err != nil {
		l.Error(
			"Failed to write response",
			"error",
			err,
		)
	}
}
