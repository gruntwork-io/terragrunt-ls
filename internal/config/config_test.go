package config_test

import (
	"terragrunt-ls/internal/config"
	"testing"

	"github.com/gruntwork-io/terragrunt/pkg/venv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadGrantsNothingByDefault pins that Load returns an empty policy when it
// gets no flags.
func TestLoadGrantsNothingByDefault(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load(nil)
	require.NoError(t, err)

	assert.Empty(t, cfg.Policy.Granted)
	assert.Empty(t, cfg.Policy.AllowCmds)
}

// TestLoadParsesAllowFlags pins that Load reads --allow both when the flag is
// repeated and when its value is comma-separated, and that it adds an
// --allow-cmd pattern to the policy.
func TestLoadParsesAllowFlags(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load([]string{"--allow=exec, env", "--allow", "sops", "--allow-cmd", "jq **"})
	require.NoError(t, err)

	assert.Equal(t, venv.Capabilities{
		venv.CapabilityExec: {},
		venv.CapabilityEnv:  {},
		venv.CapabilitySops: {},
	}, cfg.Policy.Granted)
	require.Len(t, cfg.Policy.AllowCmds, 1)
	assert.True(t, cfg.Policy.AllowCmds[0].Matches("jq", []string{"-r", ".name"}))
}

// TestLoadAcceptsTheStdioFlag pins that Load accepts --stdio, which
// vscode-languageclient adds to the arguments of a server that uses the stdio
// transport.
func TestLoadAcceptsTheStdioFlag(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load([]string{"--allow=exec", "--stdio"})
	require.NoError(t, err)

	assert.Equal(t, venv.Capabilities{venv.CapabilityExec: {}}, cfg.Policy.Granted)
}

// TestLoadRejectsInvalidAllowFlags pins the error that Load returns for each
// kind of invalid --allow or --allow-cmd value.
func TestLoadRejectsInvalidAllowFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		want error
		name string
		args []string
	}{
		{
			name: "unknown capability",
			args: []string{"--allow=exe"},
			want: venv.ErrUnknownCapability,
		},
		{
			name: "allow-cmd with a glob program",
			args: []string{"--allow=exec", "--allow-cmd", "j* **"},
			want: venv.ErrInvalidAllowCmd,
		},
		{
			name: "allow-cmd without exec",
			args: []string{"--allow-cmd", "jq **"},
			want: venv.ErrAllowCmdRequiresExec,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := config.Load(tt.args)
			require.ErrorIs(t, err, tt.want)
		})
	}
}
