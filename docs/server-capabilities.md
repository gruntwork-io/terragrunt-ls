# Server Capabilities

These are the server capabilities of the Terragrunt Language Server, as defined in the [LSP spec](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#serverCapabilities).

## TextDocumentSync

The server supports full text document sync.

Every time a document is opened or changed, the server will receive an event with the full document.

When loading a document, the server will use Terragrunt's configuration parsing to parse the HCL file, and then provide the same diagnostics that Terragrunt would provide.

The parse turns on every Terragrunt experiment. See [Experiments](./setup.md#experiments).

## HoverProvider

The server provides hover information.

When a Language Server client hovers over a token, the server will provide information about that token.

Hovering over a local variable shows its evaluated value. Hovering over the name of a Terragrunt function, or of a block or attribute at the top level of the file, shows its docs from the [Terragrunt docs](https://docs.terragrunt.com/reference/hcl/). See [Docs](./setup.md#docs) for where those come from.

Hovering over the `path` variable or one of its fields, such as `path.unit`, shows its docs and a note that the variable is experimental.

## DefinitionProvider

The server provides the ability to go to definitions.

When a Language Server client requests to go to a definition, the server will provide the location of the definition.

At the moment, the only definition target that is supported is includes. When requesting to go to the definition of an include, the server will provide the location of the included file.

## CompletionProvider

The server provides completion suggestions.

When a Language Server client requests completions for a token, the server will provide a list of suggestions.

At the moment, the only completions that are supported are the names of attributes and blocks, and the fields of the `path` variable. When requesting completions for an attribute or block name, the server will provide a list of suggestions based on the current context, each with its description from the Terragrunt docs.

After `path.`, the server suggests the fields of the `path` variable that the file can read:

- `path.unit` and `path.download` in `terragrunt.hcl`.
- `path.stack` in `terragrunt.stack.hcl`.
- All fields in other `.hcl` files, such as a `root.hcl` that a unit includes.
- No fields in `terragrunt.values.hcl`.

Terragrunt rejects `path.download` in `locals`, so the server does not suggest it in a `locals` block. The server suggests no fields in a string, a heredoc or a comment. Each suggestion is marked as experimental. A `.` starts a completion request.

## FormatProvider

The server provides the ability to format Terragrunt configuration files.

When a Language Server client requests formatting, the server will format the document and return the formatted document to the client.
