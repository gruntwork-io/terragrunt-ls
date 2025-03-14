use zed_extension_api as zed;

struct TerragruntLsExtension;

impl zed::Extension for TerragruntLsExtension {
    fn new() -> Self {
        Self
    }
    fn language_server_command(
        &mut self,
        _language_server_id: &zed_extension_api::LanguageServerId,
        worktree: &zed_extension_api::Worktree,
    ) -> zed_extension_api::Result<zed_extension_api::Command> {
        let path = worktree
            .which("terragrunt-ls")
            .ok_or_else(|| "The LSP for Terragrunt 'terragrunt-ls' is not installed".to_string())?;

        Ok(zed::Command {
            command: path,
            args: vec![],
            env: Default::default(),
        })
    }
}

zed::register_extension!(TerragruntLsExtension);
