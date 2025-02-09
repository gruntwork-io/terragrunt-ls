# Contributing

Although this is a side project for the Terragrunt maintainers, you are still advised to review [Terragrunt contributing guidelines](https://terragrunt.gruntwork.io/docs/community/contributing/) before contributing to this project.

## Dependencies

To get started with development, you will want to install [mise](https://mise.jdx.dev/getting-started.html#_1-install-mise-cli), and then run the following command:

```bash
mise install
```

This will install all the necessary dependencies for the project.

## Building

To build the project, you can run the following command:

```bash
go build
```

This will build the project and create an executable `terragrunt-ls` in the root of the project.

## Testing

To run the tests for the project, you can run the following command:

```bash
go test ./...
```

This will run all the tests for the project.

## Linting

To lint the project, you can run the following command:

```bash
golangci-lint run ./...
```

This will run the linter on the project.

## Installing

To install the project, you can run the following command:

```bash
go install
```

This will install the `terragrunt-ls` binary to your `$GOBIN`, which defaults to `$GOPATH/bin` see [GOPATH](https://go.dev/wiki/GOPATH) for more info.
