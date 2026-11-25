# Units

> Learn how to configure a Terragrunt unit in a terragrunt.hcl file

A [unit](/getting-started/terminology#unit) is configured in a `terragrunt.hcl` file in the unit’s directory, e.g.:

```hcl title="terragrunt.hcl"
include "root" {
  path = find_in_parent_folders("root.hcl")
}

dependencies {
  paths = ["../vpc", "../mysql", "../redis"]
}
```

Terragrunt also reads [JSON-serialized HCL](https://github.com/hashicorp/hcl/blob/hcl2/json/spec.md) from `terragrunt.hcl.json` files. Wherever the docs mention `terragrunt.hcl`, a `terragrunt.hcl.json` file works too.

Terragrunt uses the first of these as a unit’s configuration file:

1. The file passed to the `--config` flag.
2. The file set in the `TG_CONFIG` environment variable.
3. A `terragrunt.hcl` file in the current working directory.
4. A `terragrunt.hcl.json` file in the current working directory.

If none of them exists, Terragrunt exits with an error.

## Blocks

| Name                                                  | Description                                                                                                                    |
| ----------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| [`catalog`](/reference/hcl/blocks/catalog/)           | Configures the module catalogs that the catalog and scaffold commands read.                                                    |
| [`dependencies`](/reference/hcl/blocks/dependencies/) | Lists the units that run before this one in a run queue, without reading their outputs.                                        |
| [`dependency`](/reference/hcl/blocks/dependency/)     | Reads another unit's outputs, and runs that unit first in a run queue.                                                         |
| [`errors`](/reference/hcl/blocks/errors/)             | Retries or ignores errors from OpenTofu/Terraform commands and source downloads that match patterns you set.                   |
| [`exclude`](/reference/hcl/blocks/exclude/)           | Leaves the unit out of a run queue when a condition is true, for the commands you list.                                        |
| [`expansion`](/reference/hcl/blocks/expansion/)       | Generates one instance of a dependency, unit, or stack block per element of a collection.                                      |
| [`feature`](/reference/hcl/blocks/feature/)           | Declares a feature flag that the configuration reads as feature.\<name>.value, with a default the --feature flag can override. |
| [`generate`](/reference/hcl/blocks/generate/)         | Writes a file into the directory where Terragrunt runs OpenTofu/Terraform, e.g. a provider configuration shared by many units. |
| [`include`](/reference/hcl/blocks/include/)           | Merges another Terragrunt configuration file into this one, e.g. a root.hcl that every unit shares.                            |
| [`locals`](/reference/hcl/blocks/locals/)             | Names values that the rest of the file reads as local.\<name>.                                                                 |
| [`remote_state`](/reference/hcl/blocks/remote_state/) | Configures the OpenTofu/Terraform backend that stores a unit's state.                                                          |
| [`terraform`](/reference/hcl/blocks/terraform/)       | Configures where Terragrunt gets the OpenTofu/Terraform module and how it runs tofu or terraform.                              |
| [`engine`](/reference/hcl/blocks/engine/)             | Runs the unit's OpenTofu/Terraform commands through an IaC engine plugin.                                                      |
| [`outputs`](/reference/hcl/blocks/outputs/)           | Declares a unit's outputs and the mock values its dependents read before the unit is applied.                                  |
| [`values`](/reference/hcl/blocks/values/)             | Declares the values a unit or stack accepts, each with an optional type, default, description, and \`nullable\` setting.       |
| [`variables`](/reference/hcl/blocks/variables/)       | Sets OpenTofu/Terraform variables like the inputs attribute, and chooses how Terragrunt passes them.                           |

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

| Name                                                                                                                     | Description                                                                                                       |
| ------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------- |
| [`constraint_check`](/reference/hcl/functions/constraint_check/)                                                         | Returns whether a version satisfies a version constraint.                                                         |
| [`find_in_parent_folders`](/reference/hcl/functions/find_in_parent_folders/)                                             | Returns the absolute path of the nearest file or directory with a given name in a folder above the configuration. |
| [`get_aws_account_alias`](/reference/hcl/functions/get_aws_account_alias/)                                               | Returns the alias of the AWS account that Terragrunt's credentials belong to.                                     |
| [`get_aws_account_id`](/reference/hcl/functions/get_aws_account_id/)                                                     | Returns the ID of the AWS account that Terragrunt's credentials belong to.                                        |
| [`get_aws_caller_identity_arn`](/reference/hcl/functions/get_aws_caller_identity_arn/)                                   | Returns the ARN of the AWS identity behind Terragrunt's credentials.                                              |
| [`get_aws_caller_identity_user_id`](/reference/hcl/functions/get_aws_caller_identity_user_id/)                           | Returns the unique ID of the AWS identity behind Terragrunt's credentials.                                        |
| [`get_default_retryable_errors`](/reference/hcl/functions/get_default_retryable_errors/)                                 | Returns Terragrunt's built-in patterns for transient OpenTofu/Terraform errors.                                   |
| [`get_env`](/reference/hcl/functions/get_env/)                                                                           | Returns the value of an environment variable.                                                                     |
| [`get_original_terragrunt_dir`](/reference/hcl/functions/get_original_terragrunt_dir/)                                   | Returns the absolute path of the directory of the unit or stack file that Terragrunt is parsing.                  |
| [`get_parent_terragrunt_dir`](/reference/hcl/functions/get_parent_terragrunt_dir/)                                       | Returns the absolute path of an included file's directory.                                                        |
| [`get_path_from_repo_root`](/reference/hcl/functions/get_path_from_repo_root/)                                           | Returns the path from the root of the Git repository to the unit's directory.                                     |
| [`get_path_to_repo_root`](/reference/hcl/functions/get_path_to_repo_root/)                                               | Returns the relative path from the unit's directory to the root of the Git repository, e.g. \`../..\`.            |
| [`get_platform`](/reference/hcl/functions/get_platform/)                                                                 | Returns the operating system Terragrunt runs on, e.g. \`linux\`.                                                  |
| [`get_repo_root`](/reference/hcl/functions/get_repo_root/)                                                               | Returns the absolute path of the root of the Git repository.                                                      |
| [`get_terraform_cli_args`](/reference/hcl/functions/get_terraform_cli_args/)                                             | Returns the OpenTofu/Terraform command Terragrunt is running and its arguments.                                   |
| [`get_terraform_command`](/reference/hcl/functions/get_terraform_command/)                                               | Returns the OpenTofu/Terraform command Terragrunt is running, e.g. \`plan\`.                                      |
| [`get_terraform_commands_that_need_input`](/reference/hcl/functions/get_terraform_commands_that_need_input/)             | Returns the OpenTofu/Terraform commands that accept \`-input\`.                                                   |
| [`get_terraform_commands_that_need_locking`](/reference/hcl/functions/get_terraform_commands_that_need_locking/)         | Returns the OpenTofu/Terraform commands that accept \`-lock-timeout\`.                                            |
| [`get_terraform_commands_that_need_parallelism`](/reference/hcl/functions/get_terraform_commands_that_need_parallelism/) | Returns the OpenTofu/Terraform commands that accept \`-parallelism\`.                                             |
| [`get_terraform_commands_that_need_vars`](/reference/hcl/functions/get_terraform_commands_that_need_vars/)               | Returns the OpenTofu/Terraform commands that accept \`-var\` and \`-var-file\`.                                   |
| [`get_terragrunt_dir`](/reference/hcl/functions/get_terragrunt_dir/)                                                     | Returns the absolute path of the directory of the unit or stack file.                                             |
| [`get_terragrunt_source_cli_flag`](/reference/hcl/functions/get_terragrunt_source_cli_flag/)                             | Returns the value of the \`--source\` flag, or an empty string when it is not set.                                |
| [`get_working_dir`](/reference/hcl/functions/get_working_dir/)                                                           | Returns the absolute path of the directory where Terragrunt runs OpenTofu/Terraform for the unit.                 |
| [`mark_as_read`](/reference/hcl/functions/mark_as_read/)                                                                 | Records a file as read by the unit and returns the path unchanged.                                                |
| [`mark_glob_as_read`](/reference/hcl/functions/mark_glob_as_read/)                                                       | Records every file that matches a glob as read by the unit and returns their absolute paths.                      |
| [`path_relative_from_include`](/reference/hcl/functions/path_relative_from_include/)                                     | Returns the path from the unit's directory to an included file's directory.                                       |
| [`path_relative_to_include`](/reference/hcl/functions/path_relative_to_include/)                                         | Returns the path from an included file's directory to the unit's directory.                                       |
| [`read_terragrunt_config`](/reference/hcl/functions/read_terragrunt_config/)                                             | Parses a Terragrunt configuration file and returns its blocks and attributes.                                     |
| [`read_tfvars_file`](/reference/hcl/functions/read_tfvars_file/)                                                         | Returns the variables in a \`.tfvars\` or \`.tfvars.json\` file, encoded as JSON.                                 |
| [`run_cmd`](/reference/hcl/functions/run_cmd/)                                                                           | Runs a command and returns its stdout.                                                                            |
| [`sops_decrypt_file`](/reference/hcl/functions/sops_decrypt_file/)                                                       | Returns the decrypted content of a file encrypted with SOPS.                                                      |
| [`base64gzip_compat`](/reference/hcl/functions/base64gzip_compat/)                                                       | Returns the value \`base64gzip\` returned in Terragrunt v1.1.3 and earlier.                                       |
| [`deep_merge`](/reference/hcl/functions/deep_merge/)                                                                     | Merges maps and objects, including nested ones, into one object.                                                  |

The [Functions](/reference/hcl/functions) page documents the functions not listed here.
