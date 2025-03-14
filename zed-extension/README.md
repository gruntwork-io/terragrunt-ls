# zed-terragrunt

[Zed extension](https://zed.dev/docs/extensions/installing-extensions) for [terragrunt-ls](https://github.com/gruntwork-io/terragrunt-ls), mostly based on [terraform extension](https://github.com/zed-extensions/terraform)

## Configuration

By default this extension will recongize only `terragrunt.hcl` and `terragrunt.stack.hcl` as Terragrunt file. You can configure following setting to treat all `.hcl` as Terragrunt files:

```json
{
  "file_types": {
    "Terragrunt": ["*.hcl"]
  }
}
```
