# Setup

## Setting up build dependencies

To bootstrap your development environment, the most convenient method is to [install mise](https://mise.jdx.dev/installing-mise.html).

After installing `mise`, you can run the following command to install all necessary build dependencies for this project:

```bash
mise install
```

Alternatively, you can install the relevant dependencies manually by reading the [mise.toml](../mise.toml) file, and installing the dependencies listed there.

## Building the Language Server

To setup the language server in your editor, first install `terragrunt-ls` by running the following at the root of this repository:

```bash
go install
```

(In the future, this will be available as a precompiled binary for download)

Then follow the instructions below for your editor:

## Visual Studio Code

To install the Visual Studio Code extension, you can manually compile the extension locally, then install it from the `.vsix` file.

1. Navigate to the `vscode-extension` directory:

   ```bash
   cd vscode-extension
   ```

2. Ensure you have vsce (Visual Studio Code Extension CLI) & the typescript compiler installed. If you don't have it, you can install it globally using npm:

   ```bash
   npm install -g @vscode/vsce
   npm install -g typescript
   ```

3. Install local javascript packages

   ```bash
   npm install
   ```

4. Run the following command to package the extension:

   ```bash
   vsce package
   ```

5. This will create a `.vsix` file in the `vscode-extension` directory (e.g. `terragrunt-ls-0.0.1.vsix`). You can install this file directly as a Visual Studio Code extension, like so:

   ```bash
    code --install-extension terragrunt-ls-0.0.1.vsix
    ```

Installation from the Visual Studio Extensions Marketplace coming soon!

### Allowing capabilities in Visual Studio Code

To grant the capabilities described in [Allowing capabilities](#allowing-capabilities), set `terragrunt-ls.allow` and `terragrunt-ls.allowCmds` in your user settings:

```json
{
  "terragrunt-ls.allow": ["exec"],
  "terragrunt-ls.allowCmds": ["jq **"]
}
```

Visual Studio Code ignores both settings in a workspace's `.vscode/settings.json`, so a repository that you open cannot grant capabilities to itself.

## Neovim

For Neovim, you can install the neovim plugin by adding the following to your editor:

```lua
-- ~/.config/nvim/lua/custom/plugins/terragrunt-ls.lua

return {
  {
    "gruntwork-io/terragrunt-ls",
    -- To use a local version of the Neovim plugin, you can use something like following:
    -- dir = vim.fn.expand '~/repos/src/github.com/gruntwork-io/terragrunt-ls',
    ft = 'hcl',
    config = function()
      local terragrunt_ls = require 'terragrunt-ls'
      terragrunt_ls.setup {
        cmd_env = {
          -- If you want to see language server logs,
          -- set this to the path you want.
          -- TG_LS_LOG = vim.fn.expand '/tmp/terragrunt-ls.log',
        },
      }
      if terragrunt_ls.client then
        vim.api.nvim_create_autocmd('FileType', {
          pattern = 'hcl',
          callback = function()
            vim.lsp.buf_attach_client(0, terragrunt_ls.client)
          end,
        })
      end
    end,
  },
}
```

Installation from Mason coming soon!

### Allowing capabilities in Neovim

To grant the capabilities described in [Allowing capabilities](#allowing-capabilities), add the flags to `cmd`:

```lua
terragrunt_ls.setup {
  cmd = { 'terragrunt-ls', '--allow=exec', '--allow-cmd', 'jq **' },
}
```

## Zed

For now, clone this repo and point to the `zed-extension` directory when [installing dev extension](https://zed.dev/docs/extensions/developing-extensions#developing-an-extension-locally)

Installing from extension page coming soon!

### Allowing capabilities in Zed

To grant the capabilities described in [Allowing capabilities](#allowing-capabilities), add the flags to the language server's binary arguments in your user settings:

```json
{
  "lsp": {
    "terragrunt": {
      "binary": {
        "arguments": ["--allow=exec", "--allow-cmd", "jq **"]
      }
    }
  }
}
```

Zed also reads `lsp` settings from a project's `.zed/settings.json` once you trust that project, so a trusted project can set these flags.

## Allowing capabilities

`terragrunt-ls` parses your configuration as you edit it. By default, the parse does not run commands, make network requests, decrypt SOPS files, or read environment variables. When `terragrunt-ls` denies a command or a SOPS decryption, it reports an information diagnostic that names the flag you need.

The `--allow` flag grants a capability: `exec`, `http`, `sops`, or `env`. To grant several, repeat the flag or separate the values with commas:

```bash
terragrunt-ls --allow=exec,env --allow-cmd 'jq **'
```

`terragrunt-ls` exits with an error if `--allow` names an unknown capability, if an `--allow-cmd` pattern is invalid, or if `--allow-cmd` is used without `--allow=exec`.

To set these flags in your editor, see the section for [Visual Studio Code](#allowing-capabilities-in-visual-studio-code), [Neovim](#allowing-capabilities-in-neovim), or [Zed](#allowing-capabilities-in-zed).

Granting a capability does not confine the filesystem. A function such as `file()` reads any file that `terragrunt-ls` can read, with or without these flags.

### The `exec` capability

`exec` lets the parse start processes.

Without it, no process starts. `run_cmd()` returns empty output, and a command that Terragrunt runs on its own, such as `tofu` to read the outputs of a dependency, fails. A dependency that sets `mock_outputs` uses them.

With it, Terragrunt can run its own commands. A command that your configuration names, such as the one `run_cmd()` calls, also needs an `--allow-cmd` pattern that matches it:

```bash
# Let run_cmd() run jq with any arguments
terragrunt-ls --allow=exec --allow-cmd 'jq **'

# Let run_cmd() run only this one command line
terragrunt-ls --allow=exec --allow-cmd 'git rev-parse --show-toplevel'
```

The first word of a pattern is the program. Each word after it matches one argument, and `**` matches any number of arguments. Repeat `--allow-cmd` for more patterns. A `run_cmd()` that no pattern matches returns empty output.

A process that `exec` lets start is not restricted. It can read files, use the network, and see the environment that it is given.

### The `http` capability

`http` lets the parse make the network requests that Terragrunt makes itself. Without it, those requests fail.

### The `sops` capability

`sops` lets `sops_decrypt_file()` decrypt files. Without it, the function fails.

### The `env` capability

`env` lets the parse read the environment variables of `terragrunt-ls`, which are usually the ones your editor started it with. Without it, the parse sees an empty environment, so `get_env()` returns its default value or fails when it has none.

```bash
# Let get_env() read the environment variables of terragrunt-ls
terragrunt-ls --allow=env
```

## Docs

Hover and completion show the reference docs of Terragrunt's blocks, attributes, and functions. `terragrunt-ls` has a one-line description of each compiled in. In the background, it fetches the latest list from [docs.terragrunt.com](https://docs.terragrunt.com/reference/hcl/), and the full docs of an item the first time you hover over it. It caches what it fetches in the `terragrunt-ls/docs` directory of your user cache directory, e.g. `~/.cache/terragrunt-ls/docs` on Linux. Once a copy is an hour old, it asks the docs site whether the page changed, and downloads it again only if it did.

To stop `terragrunt-ls` from fetching docs, pass `--no-docs-fetch`. It then shows the descriptions compiled into it, and any docs it cached before.

```bash
terragrunt-ls --no-docs-fetch
```
