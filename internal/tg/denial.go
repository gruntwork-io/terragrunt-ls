package tg

import (
	"fmt"
	"regexp"
	"sync"

	lsconfig "terragrunt-ls/internal/config"

	"github.com/gruntwork-io/terragrunt/pkg/venv"
	"go.lsp.dev/protocol"
)

// maxDenialDiags bounds the diagnostics one parse reports for refusals. The
// configuration decides how many distinct refusals a parse causes, for example
// with a run_cmd() inside a for expression.
const maxDenialDiags = 25

// plainProgram matches a program name that can go in a single-quoted
// --allow-cmd suggestion as written.
var plainProgram = regexp.MustCompile(`^[A-Za-z0-9._/+-]+$`)

// denialRecorder collects the distinct refusals of one parse.
type denialRecorder struct {
	seen      map[venv.Denial]struct{}
	denials   []venv.Denial
	mu        sync.Mutex
	truncated bool
}

func newDenialRecorder() *denialRecorder {
	return &denialRecorder{seen: map[venv.Denial]struct{}{}}
}

// observe records d unless it was seen before. It is a [venv.Policy] Observe,
// so the parse may call it from several goroutines at once.
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

// diagnostics returns an Information diagnostic for each recorded refusal. They
// have no range, because a refusal does not say which call caused it.
func (r *denialRecorder) diagnostics() []protocol.Diagnostic {
	r.mu.Lock()
	defer r.mu.Unlock()

	diags := make([]protocol.Diagnostic, 0, len(r.denials)+1)
	for _, d := range r.denials {
		diags = append(diags, denialDiagnostic(denialMessage(d)))
	}

	if r.truncated {
		diags = append(diags, denialDiagnostic(fmt.Sprintf(
			"Parsing refused more than %d commands or decrypts. Only the first %d are shown.",
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

// denialMessage describes what d refused and names the flag that would allow it.
//
// Panics when d.Kind is not a [venv.DenialKind] Terragrunt defines.
func denialMessage(d venv.Denial) string {
	switch d.Kind {
	case venv.StubbedCommand:
		return fmt.Sprintf(
			"%q returned empty output, because terragrunt-ls was started without --%s=%s. "+
				"Start terragrunt-ls with --%s=%s and %s to run the command.",
			d.Cmdline,
			lsconfig.AllowFlagName, venv.CapabilityExec,
			lsconfig.AllowFlagName, venv.CapabilityExec,
			allowCmdSuggestion(d.Program),
		)
	case venv.RefusedCommand:
		return fmt.Sprintf(
			"%q returned empty output, because no --%s pattern matches it. Start terragrunt-ls with %s to run the command.",
			d.Cmdline,
			lsconfig.AllowCmdFlagName,
			allowCmdSuggestion(d.Program),
		)
	case venv.DeniedIaC, venv.DeniedTrustedCommand:
		return fmt.Sprintf(
			"%q did not run, because terragrunt-ls was started without --%s=%s.",
			d.Cmdline,
			lsconfig.AllowFlagName, venv.CapabilityExec,
		)
	case venv.DeniedSops:
		return fmt.Sprintf(
			"%q was not decrypted, because terragrunt-ls was started without --%s=%s.",
			d.Path,
			lsconfig.AllowFlagName, venv.CapabilitySops,
		)
	}

	panic(fmt.Sprintf("tg: unknown venv.DenialKind %d", d.Kind))
}

// allowCmdSuggestion returns an --allow-cmd flag that lets program run with any
// arguments, or a description of one when program cannot be quoted as written.
func allowCmdSuggestion(program string) string {
	if !plainProgram.MatchString(program) {
		return fmt.Sprintf("an --%s pattern that matches it", lsconfig.AllowCmdFlagName)
	}

	return fmt.Sprintf("--%s '%s **'", lsconfig.AllowCmdFlagName, program)
}
