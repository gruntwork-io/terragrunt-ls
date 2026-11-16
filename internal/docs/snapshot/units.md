# Units

> Learn how to configure a Terragrunt unit in a terragrunt.hcl file

A [unit](/getting-started/terminology#unit) is configured in a `terragrunt.hcl` file in the unit’s directory, e.g.:

terragrunt.hcl

```hcl
include "root" {
  path = find_in_parent_folders("root.hcl")
}

dependencies {
  paths = ["../vpc", "../mysql", "../redis"]
}
```

Terragrunt also reads [JSON-serialized HCL](https://github.com/hashicorp/hcl/blob/hcl2/json/spec.md) from `terragrunt.hcl.json` files. Wherever the docs mention `terragrunt.hcl`, a `terragrunt.hcl.json` file works too.

Terragrunt uses the first of these as a unit’s configuration file:

1. The file the `--config` flag names.
2. The file the `TG_CONFIG` environment variable names.
3. A `terragrunt.hcl` file in the current working directory.
4. A `terragrunt.hcl.json` file in the current working directory.

If none of them exists, Terragrunt exits with an error.

## Blocks

| Name                                                  | Description                                                                                       |
| ----------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| [`catalog`](/reference/hcl/blocks/catalog/)           | Configures the module catalogs that the catalog and scaffold commands read.                       |
| [`expansion`](/reference/hcl/blocks/expansion/)       | Generates one instance of a dependency, unit, or stack block per element of a collection.         |
| [`remote_state`](/reference/hcl/blocks/remote_state/) | Configures the OpenTofu/Terraform backend that stores a unit's state.                             |
| [`terraform`](/reference/hcl/blocks/terraform/)       | Configures where Terragrunt gets the OpenTofu/Terraform module and how it runs tofu or terraform. |

The [Blocks](/reference/hcl/blocks) page documents the blocks not listed here.

## Attributes

| Name                                                                                        | Description                                                                                           |
| ------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| [`download_dir`](/reference/hcl/attributes/download_dir/)                                   | Overrides the directory Terragrunt downloads OpenTofu/Terraform modules into.                         |
| [`iam_assume_role_duration`](/reference/hcl/attributes/iam_assume_role_duration/)           | Sets the STS session duration, in seconds, for the IAM role Terragrunt assumes.                       |
| [`iam_assume_role_session_name`](/reference/hcl/attributes/iam_assume_role_session_name/)   | Sets the STS session name for the IAM role Terragrunt assumes.                                        |
| [`iam_role`](/reference/hcl/attributes/iam_role/)                                           | Sets the IAM role Terragrunt assumes before invoking OpenTofu/Terraform.                              |
| [`iam_web_identity_token`](/reference/hcl/attributes/iam_web_identity_token/)               | Sets the web identity token, or the path to it, for assuming iam_role with AssumeRoleWithWebIdentity. |
| [`inputs`](/reference/hcl/attributes/inputs/)                                               | Sets the input variables Terragrunt passes to OpenTofu/Terraform.                                     |
| [`prevent_destroy`](/reference/hcl/attributes/prevent_destroy/)                             | Stops the destroy and run --all destroy commands from destroying a unit's resources.                  |
| [`terraform_binary`](/reference/hcl/attributes/terraform_binary/)                           | Overrides the OpenTofu/Terraform binary Terragrunt calls.                                             |
| [`terraform_version_constraint`](/reference/hcl/attributes/terraform_version_constraint/)   | Sets the OpenTofu/Terraform versions a unit accepts.                                                  |
| [`terragrunt_version_constraint`](/reference/hcl/attributes/terragrunt_version_constraint/) | Specifies which versions of the Terragrunt CLI can be used with your configuration.                   |

## Functions

| Name                                           | Description                                   |
| ---------------------------------------------- | --------------------------------------------- |
| [`get_env`](/reference/hcl/functions/get_env/) | Returns the value of an environment variable. |

The [Functions](/reference/hcl/functions) page documents the functions not listed here.
