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
	"go.uber.org/zap"
)

func main() {
	logfile := os.Getenv("TG_LS_LOG")

	logger := logger.NewLogger(logfile)

	l := logger.Sugar()

	defer func() {
		err := logger.Sync()
		if err != nil {
			l.Errorf("Failed to sync logger: %s", err)
		}
	}()

	l.Info("Initializing terragrunt-ls")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(rpc.Split)

	// state := analysis.NewState()
	state := tg.NewState()
	writer := os.Stdout

	for scanner.Scan() {
		msg := scanner.Bytes()

		method, contents, err := rpc.DecodeMessage(msg)
		if err != nil {
			l.Errorf("Got an error decoding the message: %w", err)
			continue
		}

		handleMessage(l, writer, state, method, contents)
	}
}

func handleMessage(l *zap.SugaredLogger, writer io.Writer, state tg.State, method string, contents []byte) {
	l.Debugf("Received msg with method: %s", method)
	l.Debugf("Contents: %s", contents)

	switch method {
	case protocol.MethodInitialize:
		var request lsp.InitializeRequest
		if err := json.Unmarshal(contents, &request); err != nil {
			l.Errorf("Failed to parse initialize request: %w", err)
		}

		l.Debugf("Connected to: %s %s",
			request.Params.ClientInfo.Name,
			request.Params.ClientInfo.Version)

		msg := lsp.NewInitializeResponse(request.ID)
		writeResponse(l, writer, msg)

		l.Debugf("Initialized")

	case protocol.MethodTextDocumentDidOpen:
		var notification lsp.DidOpenTextDocumentNotification
		if err := json.Unmarshal(contents, &notification); err != nil {
			l.Errorf("Failed to parse didOpen request: %s", err)
		}

		l.Debugf("Opened: %s", notification.Params.TextDocument.URI)

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

		l.Debug(state.Stores)

		l.Debug("Document opened")

	case protocol.MethodTextDocumentDidChange:
		var notification lsp.DidChangeTextDocumentNotification
		if err := json.Unmarshal(contents, &notification); err != nil {
			l.Errorf("Failed to parse didChange request: %w", err)
		}

		l.Debugf("Changed: %s", notification.Params.TextDocument.URI)

		for _, change := range notification.Params.ContentChanges {
			l.Debugf("Change: %s", change.Text)

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

		l.Debugf("Document changed")

	case protocol.MethodTextDocumentHover:
		var request lsp.HoverRequest
		if err := json.Unmarshal(contents, &request); err != nil {
			l.Debugf("Failed to parse hover request: %s", err)
		}

		l.Debugf("Hover: %s", request.Params.TextDocument.URI)

		response := state.Hover(l, request.ID, request.Params.TextDocument.URI, request.Params.Position)

		writeResponse(l, writer, response)

	case protocol.MethodTextDocumentDefinition:
		var request lsp.DefinitionRequest
		if err := json.Unmarshal(contents, &request); err != nil {
			l.Errorf("Failed to parse definition request: %s", err)
		}

		l.Debugf("Definition: %s", request.Params.TextDocument.URI)

		response := state.Definition(l, request.ID, request.Params.TextDocument.URI, request.Params.Position)

		writeResponse(l, writer, response)

	case protocol.MethodTextDocumentCompletion:
		var request lsp.CompletionRequest
		if err := json.Unmarshal(contents, &request); err != nil {
			l.Errorf("Failed to parse completion request: %s", err)
		}

		l.Debugf("Completion: %s", request.Params.TextDocument.URI)

		response := state.TextDocumentCompletion(l, request.ID, request.Params.TextDocument.URI, request.Params.Position)

		l.Debugf("Completion response: %v", response)

		writeResponse(l, writer, response)
	}
}

func writeResponse(l *zap.SugaredLogger, writer io.Writer, msg any) {
	reply := rpc.EncodeMessage(msg)

	_, err := writer.Write([]byte(reply))
	if err != nil {
		l.Errorf("Failed to write response: %s", err)
	}
}
