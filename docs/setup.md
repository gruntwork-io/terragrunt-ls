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

## Zed

For now, clone this repo and point to the `zed-extension` directory when [installing dev extension](https://zed.dev/docs/extensions/developing-extensions#developing-an-extension-locally)

Installing from extension page coming soon!
