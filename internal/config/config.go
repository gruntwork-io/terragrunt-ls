// Package config provides configuration loading for terragrunt-ls.
package config

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/gruntwork-io/terragrunt/pkg/venv"
)

// Config holds the configuration for terragrunt-ls
type Config struct {
	// LogFile is the path to the log file, empty string means stderr
	LogFile string
	// Policy is the set of capabilities and command patterns that the --allow
	// and --allow-cmd flags granted. The zero value grants nothing.
	Policy venv.Policy
	// LogLevel is the log level to use
	LogLevel slog.Level
}

const (
	// EnvLogFile is the environment variable that specifies the log file.
	EnvLogFile = "TG_LS_LOG"
	// EnvLogLevel is the environment variable that specifies the log level.
	EnvLogLevel = "TG_LS_LOG_LEVEL"
)

const (
	// AllowFlagName is the flag that grants a capability: exec, http, sops, or
	// env.
	AllowFlagName = "allow"
	// AllowCmdFlagName is the flag that takes a pattern for a command that
	// run_cmd() is allowed to run.
	AllowCmdFlagName = "allow-cmd"
)

const stdioFlagName = "stdio"

// Load reads configuration from environment variables and from args, which are
// the command-line arguments without the program name.
//
// Returns [flag.ErrHelp] when args contain -h or -help. Returns
// [venv.ErrUnknownCapability] when an --allow value is not a capability name,
// [venv.ErrInvalidAllowCmd] when an --allow-cmd pattern does not parse, and
// [venv.ErrAllowCmdRequiresExec] when --allow-cmd is set without --allow=exec.
func Load(args []string) (*Config, error) {
	var allow, allowCmds stringsFlag

	fs := flag.NewFlagSet("terragrunt-ls", flag.ContinueOnError)
	fs.Var(&allow, AllowFlagName,
		"Capability to grant while parsing: exec, http, sops, or env. "+
			"Repeat the flag or separate values with commas.")
	fs.Var(&allowCmds, AllowCmdFlagName,
		"Pattern, such as 'jq **', for a command that run_cmd() is allowed to run. "+
			"Repeat the flag for more patterns. Requires --allow=exec.")

	// LSP clients such as vscode-languageclient pass --stdio to select the
	// transport. The flag's value is unused because stdio is the only transport.
	fs.Bool(stdioFlagName, true, "Communicate over stdin and stdout. This is the only transport.")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	policy, err := parsePolicy(allow, allowCmds)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		LogFile:  os.Getenv(EnvLogFile),
		LogLevel: slog.LevelInfo, // default level
		Policy:   policy,
	}

	// Parse log level from environment variable
	if envLevel := os.Getenv(EnvLogLevel); envLevel != "" {
		levelVar := slog.LevelVar{}
		if err := levelVar.UnmarshalText([]byte(envLevel)); err != nil {
			slog.Error("Failed to parse log level", "error", err)
		} else {
			cfg.LogLevel = levelVar.Level()
		}
	}

	return cfg, nil
}

func parsePolicy(allow, allowCmds []string) (venv.Policy, error) {
	names := make([]string, 0, len(allow))
	for _, value := range allow {
		names = append(names, strings.Split(value, ",")...)
	}

	granted, err := venv.ParseCapabilities(names, venv.RestrictCapabilities())
	if err != nil {
		return venv.Policy{}, fmt.Errorf("--%s: %w", AllowFlagName, err)
	}

	cmds, err := venv.ParseAllowCmds(allowCmds)
	if err != nil {
		return venv.Policy{}, fmt.Errorf("--%s: %w", AllowCmdFlagName, err)
	}

	policy := venv.Policy{Granted: granted, AllowCmds: cmds}
	if err := policy.Validate(); err != nil {
		return venv.Policy{}, err
	}

	return policy, nil
}

// stringsFlag is a [flag.Value] for a flag that can be repeated. It keeps the
// value of each occurrence.
type stringsFlag []string

func (f *stringsFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *stringsFlag) Set(value string) error {
	*f = append(*f, value)

	return nil
}
