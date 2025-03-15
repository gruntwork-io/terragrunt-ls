# zed-terragrunt

[Zed extension](https://zed.dev/docs/extensions/installing-extensions) for [terragrunt-ls](https://github.com/gruntwork-io/terragrunt-ls), mostly based on [terraform extension](https://github.com/zed-extensions/terraform)

## Configuration

By default this extension will only recognize all HCL files as valid Terragrunt configuration files. You can configure the following setting to adjust this:

```json
{
  "file_types": {
    "Terragrunt": [
      "terragrunt.hcl", 
      "terragrunt.stack.hcl",
      "root.hcl"
    ]
  }
}
```
