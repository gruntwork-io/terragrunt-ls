local M = {}

M.config = {
	cmd = { "terragrunt-ls" },
	cmd_env = {},
}

local function start(bufnr)
	vim.lsp.start({
		name = "terragrunt-ls",
		cmd = M.config.cmd,
		cmd_env = M.config.cmd_env,
	}, { bufnr = bufnr })
end

-- Attaches terragrunt-ls to every hcl buffer, sharing one client across them.
function M.setup(user_config)
	M.config = vim.tbl_deep_extend("force", M.config, user_config or {})

	vim.api.nvim_create_autocmd("FileType", {
		group = vim.api.nvim_create_augroup("terragrunt-ls", {}),
		pattern = "hcl",
		callback = function(args)
			start(args.buf)
		end,
	})

	-- Buffers opened before setup ran have already had their FileType event.
	for _, bufnr in ipairs(vim.api.nvim_list_bufs()) do
		if vim.bo[bufnr].filetype == "hcl" then
			start(bufnr)
		end
	end
end

return M
