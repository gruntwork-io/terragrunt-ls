# Contributing

Although this is a side project for the Terragrunt maintainers, you are still advised to review [Terragrunt contributing guidelines](https://terragrunt.gruntwork.io/docs/community/contributing/) before contributing to this project.

## Dependencies

To get started with development, you will want to install [mise](https://mise.jdx.dev/getting-started.html#_1-install-mise-cli), and then run the following command:

```bash
mise install
```

This will install all the necessary dependencies for the project.

### Dependency updates

Dependabot opens one grouped pull request a week for each of Go modules, GitHub Actions, the VS Code extension (npm) and the Zed extension (cargo). A Terragrunt bump gets its own pull request, because it changes how the server parses files. Security updates still come one pull request per advisory.

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

The `Test` workflow runs `go test ./...` on Linux, macOS and Windows. The `Race Test` workflow runs `go test -race ./...` on Linux.

### Fuzzing

Packages that read client input have Go fuzz tests in `fuzz_test.go` files. `go test ./...` runs the seed inputs of each fuzz test as normal tests. The seeds include past crash inputs, so `go test ./...` fails if one of those crashes comes back.

To fuzz one test for a minute:

```bash
go test ./internal/rpc/ -run '^$' -fuzz '^FuzzSplit$' -fuzztime=60s
```

To fuzz every test in the module for 30 seconds each:

```bash
./.github/scripts/run-fuzz.sh
```

Set `FUZZ_TIME` to change the time per test, for example `FUZZ_TIME=2m`. A failing fuzz test does not stop the script. It runs the rest, then lists the failures and exits non-zero.

The `Fuzz` workflow runs the same script every day, on every push to `main`, and on demand. It keeps the generated corpus in the Actions cache between runs and starts a new corpus every week. Go replays the whole corpus before it fuzzes, and that replay counts toward `-fuzztime`, so a corpus that keeps growing would leave no time to fuzz. When a fuzz test fails, the workflow uploads the failing input as the `fuzz-failures` artifact.

To keep a failing input as a regression test, add it as an `f.Add` seed in that fuzz test. Put the seed and the fix in the same pull request, or `go test ./...` fails until the fix lands.

## Linting

To lint the project, you can run the following command:

```bash
golangci-lint run ./...
```

This will run the linter on the project.

## Static checks

The `Static Checks` workflow runs these commands on every pull request. `mise install` installs the tools.

```bash
codespell
go fix -diff ./...
go mod tidy -diff
git ls-files -z '*.sh' | xargs -0 -r shellcheck -x
```

`go fix -diff` and `go mod tidy -diff` print the changes they would make. Run them without `-diff` to apply the changes. On Go 1.26, `go fix -diff` exits 0 even when it prints a diff, so read its output.

## Vulnerabilities

The `Govulncheck` workflow runs `govulncheck` on every pull request and every Monday. A pull request fails only when it adds a vulnerability that the code can reach. The weekly run fails while any reachable vulnerability is left.

To check the whole module, or only what your branch adds since `origin/main`:

```bash
govulncheck ./...
./.github/scripts/govulncheck.sh origin/main
```

## Logging

The terragrunt-ls language server includes basic logging capabilities to help with development and debugging. The logging system is built on Go's structured logging (`slog`) package and provides multiple log levels and output formats.

### Configuration

Logging is configured using the `TG_LS_LOG` environment variable.

### Log Levels and Output Formats

#### File Logging

When `TG_LS_LOG` is set to a file path:

- **Format**: JSON structured logs
- **Level**: Debug and above (Debug, Info, Warn, Error)
- **Output**: Specified file

```bash
# Enable detailed file logging
export TG_LS_LOG=/tmp/terragrunt-ls.log
./terragrunt-ls
```

#### Console Logging

When `TG_LS_LOG` is not set:

- **Format**: Human-readable text
- **Level**: Info and above (Info, Warn, Error)
- **Output**: stderr

```bash
# Basic console logging
./terragrunt-ls
```

#### Log Levels

The log levels are the typical log levels you would expect.

- **Debug**: Detailed information for debugging.
- **Info**: General information about operations.
- **Warn**: Warning conditions that don't prevent operation.
- **Error**: Error conditions that may affect functionality.

Set the log level using the `TG_LS_LOG_LEVEL` environment variable.

### Development Usage

For development and debugging, it's recommended to use file logging:

```bash
# Set up file logging
export TG_LS_LOG=/tmp/terragrunt-ls-debug.log

# Build and run the language server
go build && ./terragrunt-ls

# Monitor logs in real-time
tail -f /tmp/terragrunt-ls-debug.log
```

### Log Structure

The logger uses structured logging with key-value pairs for better searchability and parsing:

```go
// Example usage in code
logger.Debug("Processing completion request",
    "uri", documentURI,
    "position", position)

logger.Error("Failed to parse request",
    "error", err,
    "method", method)
```

### Integration with the Terragrunt logger

The terragrunt-ls language server integrates with Terragrunt's internal logging system when parsing Terragrunt configuration files. This ensures consistent logging behavior and allows Terragrunt's internal operations to be observed through the language server's logging infrastructure.

When parsing Terragrunt buffers (in `internal/tg/parse.go`), a Terragrunt logger is created that:

- **Shares the same output destination** as the terragrunt-ls logger.
- **Matches the log level** by converting the terragrunt-ls logger level to Terragrunt's log level.
- **Uses JSON formatting** for structured logging.

This integration means that logging for terragrunt-ls will also include Terragrunt's internal parsing and processing logs, providing a complete view of what's happening during Terragrunt configuration analysis.

## Installing

To install the project, you can run the following command:

```bash
go install
```

This will install the `terragrunt-ls` binary to your `$GOBIN`, which defaults to `$GOPATH/bin` see [GOPATH](https://go.dev/wiki/GOPATH) for more info.
