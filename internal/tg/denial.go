package tg

import (
	"fmt"
	"regexp"
	"sync"

	lsconfig "terragrunt-ls/internal/config"

	"github.com/gruntwork-io/terragrunt/pkg/venv"
	"go.lsp.dev/protocol"
)

// maxDenialDiags is the largest number of denials that one parse reports. A
// configuration can produce any number of them.
const maxDenialDiags = 25

// plainProgram matches a program name that is safe to print inside single
// quotes in a suggested --allow-cmd flag. A name with a space or a quote in it
// would need shell escaping.
var plainProgram = regexp.MustCompile(`^[A-Za-z0-9._/+-]+$`)

// denialRecorder collects the denials from one parse. It drops repeats, and it
// drops every denial after the first maxDenialDiags.
type denialRecorder struct {
	seen      map[venv.Denial]struct{}
	denials   []venv.Denial
	mu        sync.Mutex
	truncated bool
}

func newDenialRecorder() *denialRecorder {
	return &denialRecorder{seen: map[venv.Denial]struct{}{}}
}

// observe records d. Terragrunt calls it through [venv.Policy.Observe], and may
// do so from several goroutines at once.
func (r *denialRecorder) observe(d venv.Denial) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.seen[d]; ok {
		return
	}

	if len(r.denials) == maxDenialDiags {
		r.truncated = true

		return
	}

	r.seen[d] = struct{}{}
	r.denials = append(r.denials, d)
}

// diagnostics returns one Information diagnostic for each recorded denial, and
// one more if any denials were dropped. The diagnostics have no range because a
// [venv.Denial] does not say where in the file the call is.
func (r *denialRecorder) diagnostics() []protocol.Diagnostic {
	r.mu.Lock()
	defer r.mu.Unlock()

	diags := make([]protocol.Diagnostic, 0, len(r.denials)+1)
	for _, d := range r.denials {
		diags = append(diags, denialDiagnostic(denialMessage(d)))
	}

	if r.truncated {
		diags = append(diags, denialDiagnostic(fmt.Sprintf(
			"terragrunt-ls denied more than %d commands or SOPS decryptions. Only the first %d are shown.",
			maxDenialDiags,
			maxDenialDiags,
		)))
	}

	return diags
}

func denialDiagnostic(msg string) protocol.Diagnostic {
	return protocol.Diagnostic{
		Severity: protocol.DiagnosticSeverityInformation,
		Source:   protocol.NewOptional("terragrunt-ls"),
		Message:  protocol.String(msg),
	}
}

// denialMessage returns the diagnostic text for d. The text names the command
// or file that was denied and the flags that allow it.
//
// Panics when d.Kind is not one of the [venv.DenialKind] constants.
func denialMessage(d venv.Denial) string {
	switch d.Kind {
	case venv.StubbedCommand:
		return fmt.Sprintf(
			"terragrunt-ls did not run %q and used empty output instead. "+
				"To run it, start terragrunt-ls with --%s=%s and %s.",
			d.Cmdline,
			lsconfig.AllowFlagName, venv.CapabilityExec,
			allowCmdSuggestion(d.Program),
		)
	case venv.RefusedCommand:
		return fmt.Sprintf(
			"terragrunt-ls did not run %q because no --%s pattern matches it, and used empty output instead. "+
				"To run it, start terragrunt-ls with %s.",
			d.Cmdline,
			lsconfig.AllowCmdFlagName,
			allowCmdSuggestion(d.Program),
		)
	case venv.DeniedIaC, venv.DeniedTrustedCommand:
		return fmt.Sprintf(
			"terragrunt-ls did not run %q. To run it, start terragrunt-ls with --%s=%s.",
			d.Cmdline,
			lsconfig.AllowFlagName, venv.CapabilityExec,
		)
	case venv.DeniedSops:
		return fmt.Sprintf(
			"terragrunt-ls did not decrypt %q. To decrypt it, start terragrunt-ls with --%s=%s.",
			d.Path,
			lsconfig.AllowFlagName, venv.CapabilitySops,
		)
	}

	panic(fmt.Sprintf("tg: unknown venv.DenialKind %d", d.Kind))
}

// allowCmdSuggestion returns the --allow-cmd flag that allows program with any
// arguments. When program does not match plainProgram, it describes the flag
// in words.
func allowCmdSuggestion(program string) string {
	if !plainProgram.MatchString(program) {
		return fmt.Sprintf("an --%s pattern that matches it", lsconfig.AllowCmdFlagName)
	}

	return fmt.Sprintf("--%s '%s **'", lsconfig.AllowCmdFlagName, program)
}
