-- Loads the Neovim plugin, opens a terragrunt.hcl, and waits for the server to initialize.
--
-- Usage: nvim -l .github/scripts/neovim-smoke-test.lua <path to terragrunt-ls>

local server = assert(arg[1], "usage: nvim -l neovim-smoke-test.lua <path to terragrunt-ls>")

-- Neovim reports an API it will remove through vim.deprecate, which only prints a warning.
vim.deprecate = function(name, alternative, version)
	error(string.format("%s is removed in Neovim %s; use %s", name, version, alternative), 2)
end

local notify = vim.notify
vim.notify = function(msg, level, opts)
	if level ~= nil and level >= vim.log.levels.WARN then
		error("notification: " .. msg, 2)
	end
	notify(msg, level, opts)
end

vim.opt.runtimepath:prepend(vim.uv.cwd())
require("terragrunt-ls").setup({ cmd = { server } })

local dir = vim.fn.tempname()
vim.fn.mkdir(dir, "p")
local file = dir .. "/terragrunt.hcl"
vim.fn.writefile({ "locals {", '  name = "smoke"', "}" }, file)
vim.cmd.edit(file)
vim.bo.filetype = "hcl"

local initialized = vim.wait(10000, function()
	local client = vim.lsp.get_clients({ bufnr = 0, name = "terragrunt-ls" })[1]
	return client ~= nil and client.initialized
end, 50)
if not initialized then
	error("terragrunt-ls did not attach to " .. file .. " and initialize within 10s")
end

for _, client in ipairs(vim.lsp.get_clients({ name = "terragrunt-ls" })) do
	client:stop()
end
vim.fn.delete(dir, "rf")
print("terragrunt-ls attached to terragrunt.hcl")
