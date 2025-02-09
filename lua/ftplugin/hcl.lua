-- local terragrunt_ls = require("terragrunt-ls")

-- if terragrunt_ls.client then
-- 	vim.lsp.buf_attach_client(0, terragrunt_ls.client)
-- else
-- 	local success = terragrunt_ls.setup()
-- 	if success then
-- 		vim.lsp.buf_attach_client(0, terragrunt_ls.client)
-- 	else
-- 		vim.notify(
-- 			"terragrunt-ls client was not started.  Please initialize using require('terragrunt-ls').setup({})",
-- 			"warn"
-- 		)
-- 	end
-- end

-- lua/ftplugin/hcl.lua
vim.notify("hcl.lua loaded", vim.log.levels.INFO) -- Add this line

local terragrunt_ls = require("terragrunt-ls")

if terragrunt_ls.client then
	vim.notify("terragrunt-ls client found", vim.log.levels.INFO)
	vim.lsp.buf_attach_client(0, terragrunt_ls.client)
else
	local success = terragrunt_ls.setup()
	if success then
		vim.notify("terragrunt-ls setup successful, attaching client", vim.log.levels.INFO)
		vim.lsp.buf_attach_client(0, terragrunt_ls.client)
	else
		vim.notify(
			"terragrunt-ls client was not started.  Please initialize using require('terragrunt-ls').setup({})",
			"warn"
		)
	end
end
