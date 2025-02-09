local M = {}

M.client = nil

M.config = {
	cmd = { "terragrunt-ls" },
	cmd_env = {},
}

function M.setup(user_config)
	M.config = vim.tbl_deep_extend("force", M.config, user_config or {})

	M.client = vim.lsp.start_client({
		name = "terragrunt-ls",
		cmd = M.config.cmd,
		cmd_env = M.config.cmd_env,
	})

	if not M.client then
		vim.notify("Failed to start terragrunt-ls", "error")
		return false
	end

	return true
end

return M
