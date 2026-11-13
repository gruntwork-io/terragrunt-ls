# Server Capabilities

These are the server capabilities of the Terragrunt Language Server, as defined in the [LSP spec](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#serverCapabilities).

## TextDocumentSync

The server supports full text document sync.

Every time a document is opened or changed, the server will receive an event with the full document.

When loading a document, the server will use Terragrunt's configuration parsing to parse the HCL file, and then provide the same diagnostics that Terragrunt would provide.

## HoverProvider

The server provides hover information.

When a Language Server client hovers over a token, the server will provide information about that token.

Hovering over a local variable shows its evaluated value. Hovering over the name of a Terragrunt function, or of a block or attribute at the top level of the file, shows its docs from the [Terragrunt docs](https://docs.terragrunt.com/reference/hcl/). See [Docs](./setup.md#docs) for where those come from.

## DefinitionProvider

The server provides the ability to go to definitions.

When a Language Server client requests to go to a definition, the server will provide the location of the definition.

At the moment, the only definition target that is supported is includes. When requesting to go to the definition of an include, the server will provide the location of the included file.

## CompletionProvider

The server provides completion suggestions.

When a Language Server client requests completions for a token, the server will provide a list of suggestions.

At the moment, the only completions that are supported are the names of attributes and blocks. When requesting completions for an attribute or block name, the server will provide a list of suggestions based on the current context, each with its description from the Terragrunt docs.

## FormatProvider

The server provides the ability to format Terragrunt configuration files.

When a Language Server client requests formatting, the server will format the document and return the formatted document to the client.
