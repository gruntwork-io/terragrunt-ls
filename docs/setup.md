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

By default, the language server runs no commands while it parses. [Allowing capabilities](#allowing-capabilities) describes what you can grant. Add the capabilities and command patterns to your user settings:

```json
{
  "terragrunt-ls.allow": ["exec"],
  "terragrunt-ls.allowCmds": ["jq **"]
}
```

A workspace's `.vscode/settings.json` can't set either one, so a repository can't grant itself anything.

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

By default, the language server runs no commands while it parses. [Allowing capabilities](#allowing-capabilities) describes what you can grant. Add the flags to `cmd`:

```lua
terragrunt_ls.setup {
  cmd = { 'terragrunt-ls', '--allow=exec', '--allow-cmd', 'jq **' },
}
```

## Zed

For now, clone this repo and point to the `zed-extension` directory when [installing dev extension](https://zed.dev/docs/extensions/developing-extensions#developing-an-extension-locally)

Installing from extension page coming soon!

### Allowing capabilities in Zed

By default, the language server runs no commands while it parses. [Allowing capabilities](#allowing-capabilities) describes what you can grant. Add the flags to the language server's binary arguments in your user settings:

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

Zed also reads `lsp` settings from a project's `.zed/settings.json` once you trust that project.

## Allowing capabilities

By default, `terragrunt-ls` parses a configuration without running commands, making network requests, decrypting SOPS files, or reading environment variables. A `run_cmd()` returns empty output, `get_env()` returns its default, and `sops_decrypt_file()` fails. Each refused command or decrypt appears as an information diagnostic that names the flag that would allow it.

Grant more with these flags:

- `--allow` grants a capability: `exec`, `http`, `sops`, or `env`. Repeat it or separate values with commas.
- `--allow-cmd` lets parsing run a program a configuration names, given as a pattern such as `'jq **'`. The first word is the program and each later word matches one argument, where `**` matches any number of arguments. It requires `--allow=exec`.

```bash
# get_env() reads the editor's environment
terragrunt-ls --allow=env

# run_cmd() may run jq with any arguments
terragrunt-ls --allow=exec --allow-cmd 'jq **'
```

`terragrunt-ls` refuses to start when a flag names an unknown capability, when a pattern can't be read, or when `--allow-cmd` is given without `--allow=exec`.

Each editor section above shows where to set these flags: [Visual Studio Code](#allowing-capabilities-in-visual-studio-code), [Neovim](#allowing-capabilities-in-neovim), and [Zed](#allowing-capabilities-in-zed).
