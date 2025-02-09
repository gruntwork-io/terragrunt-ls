# Setup

To setup the language server in your editor, first install `terragrunt-ls` by running the following at the root of this repository:

```bash
go install
```

(In the future, this will be available as a precompiled binary for download)

Then follow the instructions below for your editor:

## Visual Studio Code

Coming soon!

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
