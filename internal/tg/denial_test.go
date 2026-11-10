package tg_test

import (
	"io"
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

// TestParseTerragruntBufferReportsADeniedRunCmd pins that a run_cmd() denied by
// the policy returns empty output, and that the parse reports a diagnostic
// naming the flags that allow the command.
func TestParseTerragruntBufferReportsADeniedRunCmd(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		message string
		policy  venv.Policy
	}{
		{
			name:   "without exec",
			policy: venv.Policy{},
			message: `terragrunt-ls did not run "jq -n 1" and used empty output instead. ` +
				`To run it, start terragrunt-ls with --allow=exec and --allow-cmd 'jq **'.`,
		},
		{
			name:   "with exec and no matching allow-cmd",
			policy: venv.Policy{Granted: venv.Capabilities{venv.CapabilityExec: {}}},
			message: `terragrunt-ls did not run "jq -n 1" because no --allow-cmd pattern matches it, ` +
				`and used empty output instead. To run it, start terragrunt-ls with --allow-cmd 'jq **'.`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v := venv.MemVenv()
			filename := filepath.Join(testutils.WorkDir(t, v), "terragrunt.hcl")
			content := `locals {
  out = run_cmd("--terragrunt-quiet", "jq", "-n", "1")
}
`

			cfg, diags := tg.ParseTerragruntBuffer(
				t.Context(), testutils.NewTestLogger(t), v, tt.policy, filename, content,
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

// TestParseTerragruntBufferRunsAnAllowedRunCmd pins that a run_cmd() runs when
// an --allow-cmd pattern matches it, and that the parse reports no diagnostic
// for it.
func TestParseTerragruntBufferRunsAnAllowedRunCmd(t *testing.T) {
	t.Parallel()

	allowCmds, err := venv.ParseAllowCmds([]string{"go env GOOS"})
	require.NoError(t, err)

	// Looking up go needs PATH, and the venv has PATH only with the env
	// capability.
	policy := venv.Policy{
		Granted:   venv.Capabilities{venv.CapabilityExec: {}, venv.CapabilityEnv: {}},
		AllowCmds: allowCmds,
	}
	filename := filepath.Join(t.TempDir(), "terragrunt.hcl")
	content := `locals {
  goos = run_cmd("--terragrunt-quiet", "go", "env", "GOOS")
}
`

	// pkg/venv does not export the types that a MemVenv command handler needs,
	// so this test runs the real go binary.
	v := venv.OSVenv().WithWriter(io.Discard).WithErrWriter(io.Discard)

	cfg, diags := tg.ParseTerragruntBuffer(
		t.Context(), testutils.NewTestLogger(t), v, policy, filename, content,
	)

	require.NotNil(t, cfg)
	assert.Equal(t, runtime.GOOS, cfg.Locals["goos"])
	assert.Empty(t, diags)
}

// TestParseTerragruntBufferCapsDenialDiagnostics pins that a parse with 30
// distinct denials reports the first 25, followed by one diagnostic saying that
// the rest are not shown.
func TestParseTerragruntBufferCapsDenialDiagnostics(t *testing.T) {
	t.Parallel()

	v := venv.MemVenv()
	filename := filepath.Join(testutils.WorkDir(t, v), "terragrunt.hcl")
	content := `locals {
  outs = [for i in range(30) : run_cmd("--terragrunt-quiet", "jq", "-n", tostring(i))]
}
`

	_, diags := tg.ParseTerragruntBuffer(
		t.Context(), testutils.NewTestLogger(t), v, venv.Policy{}, filename, content,
	)

	require.Len(t, diags, 26)
	assert.Equal(
		t,
		protocol.String("terragrunt-ls denied more than 25 commands or SOPS decryptions. Only the first 25 are shown."),
		diags[25].Message,
	)
}
