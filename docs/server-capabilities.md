# Server Capabilities

These are the server capabilities of the Terragrunt Language Server, as defined in the [LSP spec](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#serverCapabilities).

## TextDocumentSync

The server supports full text document sync.

Every time a document is opened or changed, the server will receive an event with the full document.

When loading a document, the server will use Terragrunt's configuration parsing to parse the HCL file, and then provide the same diagnostics that Terragrunt would provide.

## HoverProvider

The server provides hover information.

When a user hovers over a token, the server will provide information about that token.

At the moment, the only hover target that is supported is local variables. When hovering over a local variable, the server will provide the evaluated value of that local.

## DefinitionProvider

The server provides the ability to go to defintions.

When a user requests to go to a definition, the server will provide the location of the definition.

At the moment, the only definition target that is supported is includes. When requesting to go to the definition of an include, the server will provide the location of the included file.

## CompletionProvider

The server provides completion suggestions.

When a user requests completions for a token, the server will provide a list of suggestions.

At the moment, the only completions that are supported are the names of attributes and blocks. When requesting completions for an attribute or block name, the server will provide a list of suggestions based on the current context.
