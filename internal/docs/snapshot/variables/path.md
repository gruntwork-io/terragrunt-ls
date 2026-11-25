# path

> Holds the absolute paths of the unit, the stack and the files that Terragrunt evaluates.

## Usage

Experimental

The `path` variable needs the [`path-variable`](/reference/experiments/active#path-variable) experiment. Enable it with `--experiment path-variable` or `TG_EXPERIMENT=path-variable`. Without the experiment, a reference to `path` fails with an error.

Each field of `path` is the absolute path of one location. `path.unit` and `path.download` have the same value in every file that Terragrunt evaluates for a unit. `path.include` and `path.read` name the file that Terragrunt evaluates.

A field is defined only where its location exists. A reference to a field that is not defined returns an error. Use `try` to supply a fallback:

```hcl
# common.hcl, which a unit includes or reads
locals {
  common_dir = dirname(try(path.include, path.read))
}
```

An `autoinclude` block is a part of `terragrunt.stack.hcl`. Terragrunt resolves `path` in that block when it generates the stack, so `path.stack` is defined there, and `path.unit` and `path.download` are not.

## Fields

- [`path.download`](/reference/hcl/variables/path/download/): The absolute path of the directory in which Terragrunt runs OpenTofu/Terraform for the unit.
- [`path.include`](/reference/hcl/variables/path/include/): The absolute path of the included file, e.g. `/path/to/root.hcl`.
- [`path.read`](/reference/hcl/variables/path/read/): The absolute path of the file that `read_terragrunt_config` reads, e.g. `/path/to/env.hcl`.
- [`path.stack`](/reference/hcl/variables/path/stack/): The absolute path of the directory of the `terragrunt.stack.hcl` that Terragrunt evaluates.
- [`path.unit`](/reference/hcl/variables/path/unit/): The absolute path of the directory of the unit, which holds its `terragrunt.hcl`.
