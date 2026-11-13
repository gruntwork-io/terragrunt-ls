# Stacks

> Learn how to configure a Terragrunt stack in a terragrunt.stack.hcl file

A [stack](/getting-started/terminology#stack) is a directory of units. A `terragrunt.stack.hcl` file in the stack’s directory defines the units for Terragrunt to generate: where each unit’s configuration comes from, where in the stack it goes, and the values it receives. For an introduction to stacks, see the [Stacks](/features/stacks) guide.

## Unit and stack blocks

A `unit` block generates a unit. A `stack` block generates a nested stack, whose own `terragrunt.stack.hcl` file generates more units:

terragrunt.stack.hcl

```hcl
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

terragrunt.stack.hcl

```hcl
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

### Reusing a stack

Two `stack` blocks can share a source and pass it different values:

terragrunt.stack.hcl

```hcl
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

stacks/environment/terragrunt.stack.hcl

```hcl
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

| Name                                                | Description                                                                                |
| --------------------------------------------------- | ------------------------------------------------------------------------------------------ |
| [`autoinclude`](/reference/hcl/blocks/autoinclude/) | Adds dependencies and other configuration to a unit or stack when Terragrunt generates it. |
| [`expansion`](/reference/hcl/blocks/expansion/)     | Generates one instance of a dependency, unit, or stack block per element of a collection.  |
| [`stack`](/reference/hcl/blocks/stack/)             | Defines a nested stack that Terragrunt generates from a stack file.                        |
| [`unit`](/reference/hcl/blocks/unit/)               | Defines a unit that Terragrunt generates from a stack file.                                |

A `terragrunt.stack.hcl` file also accepts [`include`](/reference/hcl/blocks#includes-in-stack-files) and [`locals`](/reference/hcl/blocks#locals) blocks.

## Functions

| Name                                           | Description                                   |
| ---------------------------------------------- | --------------------------------------------- |
| [`get_env`](/reference/hcl/functions/get_env/) | Returns the value of an environment variable. |

The [Functions](/reference/hcl/functions) page documents the functions not listed here.
