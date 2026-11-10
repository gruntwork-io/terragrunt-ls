package tg_test

import (
	"path/filepath"
	"runtime"
	"terragrunt-ls/internal/testutils"
	"terragrunt-ls/internal/tg"
	"testing"

	"github.com/gruntwork-io/terragrunt/pkg/venv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
)

// TestParseTerragruntBufferReportsARefusedRunCmd pins that a run_cmd() the
// policy refuses returns empty output, and that the parse reports it with the
// flags that would let it run.
func TestParseTerragruntBufferReportsARefusedRunCmd(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		message string
		policy  venv.Policy
	}{
		{
			name:   "without exec",
			policy: venv.Policy{},
			message: `"jq -n 1" returned empty output, because terragrunt-ls was started without --allow=exec. ` +
				`Start terragrunt-ls with --allow=exec and --allow-cmd 'jq **' to run the command.`,
		},
		{
			name:   "with exec, unallowed",
			policy: venv.Policy{Granted: venv.Capabilities{venv.CapabilityExec: {}}},
			message: `"jq -n 1" returned empty output, because no --allow-cmd pattern matches it. ` +
				`Start terragrunt-ls with --allow-cmd 'jq **' to run the command.`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			filename := filepath.Join(t.TempDir(), "terragrunt.hcl")
			content := `locals {
  out = run_cmd("--terragrunt-quiet", "jq", "-n", "1")
}
`

			cfg, diags := tg.ParseTerragruntBuffer(
				t.Context(), testutils.NewTestLogger(t), testutils.NewVenv(), tt.policy, filename, content,
			)

			require.NotNil(t, cfg)
			assert.Empty(t, cfg.Locals["out"])
			assert.Equal(t, []protocol.Diagnostic{{
				Severity: protocol.DiagnosticSeverityInformation,
				Source:   protocol.NewOptional("terragrunt-ls"),
				Message:  protocol.String(tt.message),
			}}, diags)
		})
	}
}

// TestParseTerragruntBufferRunsAnAllowedRunCmd pins that a run_cmd() an
// --allow-cmd pattern matches runs, and that the parse reports nothing for it.
func TestParseTerragruntBufferRunsAnAllowedRunCmd(t *testing.T) {
	t.Parallel()

	allowCmds, err := venv.ParseAllowCmds([]string{"go env GOOS"})
	require.NoError(t, err)

	// The env capability keeps PATH, which finding go needs.
	policy := venv.Policy{
		Granted:   venv.Capabilities{venv.CapabilityExec: {}, venv.CapabilityEnv: {}},
		AllowCmds: allowCmds,
	}
	filename := filepath.Join(t.TempDir(), "terragrunt.hcl")
	content := `locals {
  goos = run_cmd("--terragrunt-quiet", "go", "env", "GOOS")
}
`

	cfg, diags := tg.ParseTerragruntBuffer(
		t.Context(), testutils.NewTestLogger(t), testutils.NewVenv(), policy, filename, content,
	)

	require.NotNil(t, cfg)
	assert.Equal(t, runtime.GOOS, cfg.Locals["goos"])
	assert.Empty(t, diags)
}

// TestParseTerragruntBufferBoundsRefusalDiagnostics pins that a configuration
// causing more distinct refusals than the parse reports gets the first 25 and
// one diagnostic saying the rest were left out.
func TestParseTerragruntBufferBoundsRefusalDiagnostics(t *testing.T) {
	t.Parallel()

	filename := filepath.Join(t.TempDir(), "terragrunt.hcl")
	content := `locals {
  outs = [for i in range(30) : run_cmd("--terragrunt-quiet", "jq", "-n", tostring(i))]
}
`

	_, diags := tg.ParseTerragruntBuffer(
		t.Context(), testutils.NewTestLogger(t), testutils.NewVenv(), venv.Policy{}, filename, content,
	)

	require.Len(t, diags, 26)
	assert.Equal(
		t,
		protocol.String("Parsing refused more than 25 commands or decrypts. Only the first 25 are shown."),
		diags[25].Message,
	)
}
