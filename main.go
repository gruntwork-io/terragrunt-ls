package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"terragrunt-ls/internal/config"
	"terragrunt-ls/internal/logger"
	"terragrunt-ls/internal/lsp"
	"terragrunt-ls/internal/rpc"
	"terragrunt-ls/internal/tg"

	"github.com/gruntwork-io/terragrunt/pkg/venv"
	"go.lsp.dev/protocol"
)

// exitCodeUsage is the exit code for invalid flags. The flag package uses the
// same code under [flag.ExitOnError].
const exitCodeUsage = 2

func main() {
	cfg, err := config.Load(os.Args[1:])
	if errors.Is(err, flag.ErrHelp) {
		return
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCodeUsage)
	}

	l := logger.NewLogger(cfg.LogFile, cfg.LogLevel)
	defer func() {
		if err := l.Close(); err != nil {
			panic(err)
		}
	}()

	l.Info("Initializing terragrunt-ls")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(rpc.Split)

	ctx := context.Background()
	state := tg.NewState()
	writer := os.Stdout
	// The client reads LSP messages from stdout, so output that Terragrunt
	// writes while parsing is discarded.
	v := venv.OSVenv().WithWriter(io.Discard).WithErrWriter(io.Discard)

	for scanner.Scan() {
		msg := scanner.Bytes()

		method, contents, err := rpc.DecodeMessage(msg)
		if err != nil {
			l.Error("Got an error decoding message from client", "err", err)

			continue
		}

		handleMessage(ctx, l, v, cfg.Policy, writer, state, method, contents)
	}
}

func handleMessage(
	ctx context.Context,
	l logger.Logger,
	v *venv.Venv,
	p venv.Policy,
	writer io.Writer,
	state tg.State,
	method string,
	contents []byte,
) {
	l.Debug("Received msg", "method", method, "contents", string(contents))

	switch method {
	case protocol.MethodInitialize:
		var request lsp.InitializeRequest
		if err := protocol.Unmarshal(contents, &request); err != nil {
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
		if err := protocol.Unmarshal(contents, &notification); err != nil {
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

		diagnostics := state.OpenDocument(
			ctx,
			l,
			v,
			p,
			notification.Params.TextDocument.URI,
			notification.Params.TextDocument.Text,
		)
		writeResponse(l, writer, lsp.PublishDiagnosticsNotification{
			RPC:    lsp.RPCVersion,
			Method: protocol.MethodTextDocumentPublishDiagnostics,
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
		if err := protocol.Unmarshal(contents, &notification); err != nil {
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
			text, ok := lsp.ContentChangeText(change)
			if !ok {
				l.Error(
					"Ignoring null content change",
					"URI", notification.Params.TextDocument.URI,
				)

				continue
			}

			l.Debug(
				"Change",
				"Text", text,
			)

			diagnostics := state.UpdateDocument(ctx, l, v, p, notification.Params.TextDocument.URI, text)
			writeResponse(l, writer, lsp.PublishDiagnosticsNotification{
				RPC:    lsp.RPCVersion,
				Method: protocol.MethodTextDocumentPublishDiagnostics,
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
		if err := protocol.Unmarshal(contents, &request); err != nil {
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
		if err := protocol.Unmarshal(contents, &request); err != nil {
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
		if err := protocol.Unmarshal(contents, &request); err != nil {
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

	case protocol.MethodTextDocumentFormatting:
		var request lsp.FormatRequest
		if err := protocol.Unmarshal(contents, &request); err != nil {
			l.Error(
				"Failed to parse format request",
				"error",
				err,
			)
		}

		l.Debug(
			"Formatting",
			"URI", request.Params.TextDocument.URI,
		)

		response := state.TextDocumentFormatting(l, request.ID, request.Params.TextDocument.URI)

		writeResponse(l, writer, response)

	case protocol.MethodTextDocumentReferences:
		var request lsp.ReferencesRequest
		if err := protocol.Unmarshal(contents, &request); err != nil {
			l.Error(
				"Failed to parse references request",
				"error",
				err,
			)
		}

		l.Debug(
			"References",
			"URI", request.Params.TextDocument.URI,
			"Position", request.Params.Position,
			"IncludeDeclaration", request.Params.Context.IncludeDeclaration,
		)

		response := state.TextDocumentReferences(
			l,
			request.ID,
			request.Params.TextDocument.URI,
			request.Params.Position,
			request.Params.Context.IncludeDeclaration,
		)

		writeResponse(l, writer, response)

	case protocol.MethodTextDocumentPrepareRename:
		var request lsp.PrepareRenameRequest
		if err := protocol.Unmarshal(contents, &request); err != nil {
			l.Error(
				"Failed to parse prepare rename request",
				"error",
				err,
			)
		}

		l.Debug(
			"Prepare rename",
			"URI", request.Params.TextDocument.URI,
			"Position", request.Params.Position,
		)

		response := state.PrepareRename(l, request.ID, request.Params.TextDocument.URI, request.Params.Position)

		writeResponse(l, writer, response)

	case protocol.MethodTextDocumentRename:
		var request lsp.RenameRequest
		if err := protocol.Unmarshal(contents, &request); err != nil {
			l.Error(
				"Failed to parse rename request",
				"error",
				err,
			)
		}

		l.Debug(
			"Rename",
			"URI", request.Params.TextDocument.URI,
			"Position", request.Params.Position,
			"NewName", request.Params.NewName,
		)

		response := state.TextDocumentRename(
			l,
			request.ID,
			request.Params.TextDocument.URI,
			request.Params.Position,
			request.Params.NewName,
		)

		writeResponse(l, writer, response)
	}
}

func writeResponse(l logger.Logger, writer io.Writer, msg any) {
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
