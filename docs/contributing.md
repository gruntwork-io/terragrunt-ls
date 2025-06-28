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
