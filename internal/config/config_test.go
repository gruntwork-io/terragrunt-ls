package config_test

import (
	"terragrunt-ls/internal/config"
	"testing"

	"github.com/gruntwork-io/terragrunt/pkg/venv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadGrantsNothingByDefault pins that Load without flags returns a policy
// that grants no capability and allows no program.
func TestLoadGrantsNothingByDefault(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load(nil)
	require.NoError(t, err)

	assert.Empty(t, cfg.Policy.Granted)
	assert.Empty(t, cfg.Policy.AllowCmds)
}

// TestLoadGrantsWhatTheAllowFlagsName pins that --allow takes repeated and
// comma-separated capabilities, and that the policy has the --allow-cmd
// patterns.
func TestLoadGrantsWhatTheAllowFlagsName(t *testing.T) {
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

// TestLoadRejectsFlagsItCannotHonor pins that Load fails on an --allow or
// --allow-cmd value that would grant less than the operator asked for.
func TestLoadRejectsFlagsItCannotHonor(t *testing.T) {
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
