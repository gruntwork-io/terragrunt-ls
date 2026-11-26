# Stacks

> Learn how to configure a Terragrunt stack in a terragrunt.stack.hcl file

A [stack](/getting-started/terminology#stack) is a directory of units. A `terragrunt.stack.hcl` file in the stack’s directory defines the units for Terragrunt to generate: where each unit’s configuration comes from, where in the stack it goes, and the values it receives. For an introduction to stacks, see the [Stacks](/features/stacks) guide.

## Unit and stack blocks

A `unit` block generates a unit. A `stack` block generates a nested stack, whose own `terragrunt.stack.hcl` file generates more units:

```hcl title="terragrunt.stack.hcl"
unit "vpc" {
  source = "../catalog/units/vpc"
  path   = "vpc"
}

stack "dev" {
  source = "../catalog/stacks/environment"
  path   = "dev"
}
```

Running `terragrunt stack generate` creates:

* terragrunt.stack.hcl

* .terragrunt-stack

  * vpc

    * **terragrunt.hcl** from `unit "vpc"`

  * dev

    * **terragrunt.stack.hcl** from `stack "dev"`

    * .terragrunt-stack

      * vpc

        * terragrunt.hcl

      * database

        * terragrunt.hcl

Use a `stack` block for a group of units you deploy more than once, e.g. one per environment.

## Generating units

[`terragrunt stack generate`](/reference/cli/commands/stack/generate) writes the units and stacks that a `terragrunt.stack.hcl` file defines, by default under a `.terragrunt-stack` directory. [`stack run`](/reference/cli/commands/stack/run) and `run --all` generate them before they run, so `terragrunt stack run apply` generates and deploys a stack in one step.

A generated unit is an ordinary unit configuration, and Terragrunt reads it as if you had written it.

## Examples

### Passing values to units

```hcl title="terragrunt.stack.hcl"
unit "vpc" {
  source = "../catalog/units/vpc"
  path   = "vpc"
  values = {
    vpc_name = "main"
    cidr     = "10.0.0.0/16"
  }
}

unit "database" {
  source = "../catalog/units/database"
  path   = "database"
  values = {
    engine   = "postgres"
    version  = "13"
    vpc_path = "../vpc"
  }
}
```

Terragrunt writes each unit’s `values` to a `terragrunt.values.hcl` file next to its `terragrunt.hcl` file:

* terragrunt.stack.hcl

* .terragrunt-stack

  * vpc

    * terragrunt.hcl
    * **terragrunt.values.hcl**

  * database

    * terragrunt.hcl
    * **terragrunt.values.hcl**

The unit’s `terragrunt.hcl` file reads them as `values.<name>`, e.g. `values.cidr`.

With the [`values-block`](/reference/experiments/active#values-block) experiment, a unit can declare the values it reads in a [`values`](/reference/hcl/blocks/values) block, each with an optional type and default.

### Reusing a stack

Two `stack` blocks can share a source and pass it different values:

```hcl title="terragrunt.stack.hcl"
stack "dev" {
  source = "git::git@github.com:acme/infrastructure-catalog.git//stacks/environment?ref=v0.0.1"
  path   = "dev"
  values = {
    environment = "development"
    cidr        = "10.0.0.0/16"
  }
}

stack "prod" {
  source = "git::git@github.com:acme/infrastructure-catalog.git//stacks/environment?ref=v0.0.1"
  path   = "prod"
  values = {
    environment = "production"
    cidr        = "10.1.0.0/16"
  }
}
```

The stack at that source reads them the same way:

```hcl title="stacks/environment/terragrunt.stack.hcl"
unit "vpc" {
  source = "git::git@github.com:acme/infrastructure-catalog.git//units/vpc?ref=v0.0.1"
  path   = "vpc"
  values = {
    vpc_name = values.environment
    cidr     = values.cidr
  }
}

unit "database" {
  source = "git::git@github.com:acme/infrastructure-catalog.git//units/database?ref=v0.0.1"
  path   = "database"
  values = {
    environment = values.environment

    vpc_path = "../vpc"
  }
}
```

## Blocks

| Name                                                | Description                                                                                                              |
| --------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| [`autoinclude`](/reference/hcl/blocks/autoinclude/) | Adds dependencies and other configuration to a unit or stack when Terragrunt generates it.                               |
| [`expansion`](/reference/hcl/blocks/expansion/)     | Generates one instance of a dependency, unit, or stack block per element of a collection.                                |
| [`include`](/reference/hcl/blocks/include/)         | Merges another Terragrunt configuration file into this one, e.g. a root.hcl that every unit shares.                      |
| [`locals`](/reference/hcl/blocks/locals/)           | Names values that the rest of the file reads as local.\<name>.                                                           |
| [`stack`](/reference/hcl/blocks/stack/)             | Defines a nested stack that Terragrunt generates from a stack file.                                                      |
| [`unit`](/reference/hcl/blocks/unit/)               | Defines a unit that Terragrunt generates from a stack file.                                                              |
| [`values`](/reference/hcl/blocks/values/)           | Declares the values a unit or stack accepts, each with an optional type, default, description, and \`nullable\` setting. |

A `terragrunt.stack.hcl` file also accepts [`include`](/reference/hcl/blocks/include#includes-in-stack-files) and [`locals`](/reference/hcl/blocks/locals) blocks.

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
