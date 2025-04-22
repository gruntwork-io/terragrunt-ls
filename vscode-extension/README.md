# Language Server for Terragrunt

This is the official Language Server for [Terragrunt](https://terragrunt.gruntwork.io/).

## Functionality

See the [Language Server README](https://github.com/gruntwork-io/terragrunt-ls) for a full list of features.

Some highlights:

### TextDocumentSync

The server supports full text document sync.

Every time a document is opened or changed, the server will receive an event with the full document.

When loading a document, the server will use Terragrunt's configuration parsing to parse the HCL file, and then provide the same diagnostics that Terragrunt would provide.

### HoverProvider

The server provides hover information.

When a Language Server client hovers over a token, the server will provide information about that token.

At the moment, the only hover target that is supported is local variables. When hovering over a local variable, the server will provide the evaluated value of that local.

### DefinitionProvider

The server provides the ability to go to definitions.

When a Language Server client requests to go to a definition, the server will provide the location of the definition.

At the moment, the only definition target that is supported is includes. When requesting to go to the definition of an include, the server will provide the location of the included file.

### CompletionProvider

The server provides completion suggestions.

When a Language Server client requests completions for a token, the server will provide a list of suggestions.

At the moment, the only completions that are supported are the names of attributes and blocks. When requesting completions for an attribute or block name, the server will provide a list of suggestions based on the current context.

### FormatProvider

The server provides the ability to format Terragrunt configuration files.

When a Language Server client requests formatting, the server will format the document and return the formatted document to the client.

<!-- This README.md is displayed in the extension installation page, so try to keep the docs useful when user facing. -->

## Development

If you are reading this in the Git repository, you can find instructions on how to set up your local development environment and run the extension in development mode in the [Development README](./docs/development.md).
