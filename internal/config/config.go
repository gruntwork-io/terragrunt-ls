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
	// Policy is what parsing a configuration may do beyond reading files.
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
	// AllowFlagName is the name of the flag that grants a capability parsing
	// otherwise refuses.
	AllowFlagName = "allow"
	// AllowCmdFlagName is the name of the flag that lets parsing run a program
	// a configuration names, matched on its program and arguments.
	AllowCmdFlagName = "allow-cmd"
)

// Load reads configuration from environment variables and from args, the
// command-line arguments without the program name.
//
// Returns [flag.ErrHelp] when args ask for help, [venv.ErrUnknownCapability]
// for an --allow value that names no capability, [venv.ErrInvalidAllowCmd] for
// an --allow-cmd pattern it cannot read, and [venv.ErrAllowCmdRequiresExec] for
// --allow-cmd without --allow=exec.
func Load(args []string) (*Config, error) {
	var allow, allowCmds stringsFlag

	fs := flag.NewFlagSet("terragrunt-ls", flag.ContinueOnError)
	fs.Var(&allow, AllowFlagName,
		"Grant a capability parsing otherwise refuses: exec, http, sops, or env. Repeatable, or comma-separated.")
	fs.Var(&allowCmds, AllowCmdFlagName,
		"Let parsing run a program a configuration names, such as one run_cmd calls, given as a pattern like 'jq **'. "+
			"Repeatable. Requires --allow=exec.")

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

// parsePolicy builds the policy the --allow and --allow-cmd values grant.
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

// stringsFlag is a [flag.Value] that keeps every value it is given, so its
// flag can repeat.
type stringsFlag []string

func (f *stringsFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *stringsFlag) Set(value string) error {
	*f = append(*f, value)

	return nil
}
