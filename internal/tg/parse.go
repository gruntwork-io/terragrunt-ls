package tg

import (
	"context"
	"io"
	"path/filepath"
	"strings"

	"terragrunt-ls/internal/logger"
	"terragrunt-ls/internal/tg/store"

	"github.com/gruntwork-io/terragrunt/pkg/config"
	"github.com/gruntwork-io/terragrunt/pkg/config/hclparse"
	tgLog "github.com/gruntwork-io/terragrunt/pkg/log"
	"github.com/gruntwork-io/terragrunt/pkg/log/format"
	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"
	"go.lsp.dev/protocol"
)

const defaultMaxFoldersToCheck = 100

// parsingSetup holds the shared state for parsing terragrunt config files.
type parsingSetup struct {
	pctx       *config.ParsingContext
	parseDiags hcl.Diagnostics
}

func newTGLogger(l logger.Logger) tgLog.Logger {
	return tgLog.New(
		tgLog.WithOutput(l.Writer()),
		tgLog.WithLevel(tgLog.FromLogrusLevel(logrus.Level(l.Level()))),
		tgLog.WithFormatter(format.NewFormatter(format.NewJSONFormatPlaceholders())),
	)
}

func newParsingSetup(ctx context.Context, tgLogger tgLog.Logger, filename string) (context.Context, *parsingSetup) {
	s := &parsingSetup{}

	parseOptions := []hclparse.Option{
		hclparse.WithDiagnosticsHandler(func(file *hcl.File, hclDiags hcl.Diagnostics) (hcl.Diagnostics, error) {
			s.parseDiags = append(s.parseDiags, hclDiags...)
			return hclDiags, nil
		}),
	}

	ctx, s.pctx = config.NewParsingContext(ctx, tgLogger)
	s.pctx.TerragruntConfigPath = filename
	s.pctx.WorkingDir = filepath.Dir(filename)
	s.pctx.SkipOutput = true
	s.pctx.MaxFoldersToCheck = defaultMaxFoldersToCheck
	s.pctx.Writers.Writer = io.Discard
	s.pctx.Writers.ErrWriter = io.Discard
	s.pctx.ParserOptions = append(s.pctx.ParserOptions, parseOptions...)

	return ctx, s
}

func ParseTerragruntBuffer(ctx context.Context, l logger.Logger, filename, text string) (*config.TerragruntConfig, []protocol.Diagnostic) {
	tgLogger := newTGLogger(l)
	ctx, s := newParsingSetup(ctx, tgLogger, filename)

	cfg, err := config.ParseConfigString(ctx, s.pctx, tgLogger, filename, text, nil)
	if err != nil {
		// Just log the error for now
		l.Error("Error parsing Terragrunt config", "error", err)
	}

	filteredDiags := filterHCLDiags(l, s.parseDiags, filename, text)

	diags := hclDiagsToLSPDiags(filteredDiags)

	return cfg, diags
}

func filterHCLDiags(l logger.Logger, diags hcl.Diagnostics, filename, text string) hcl.Diagnostics {
	filtered := hcl.Diagnostics{}

	for _, diag := range diags {
		if diag.Subject == nil {
			filtered = append(filtered, diag)

			continue
		}

		l.Debug(
			"Checking to see diag can be filtered.",
			"diag", diag,
			"filename", filename,
		)

		if isMissingOutputDiag(diag) {
			l.Debug(
				"Filtering output missing diag",
				"diag", diag,
				"filename", filename,
			)

			continue
		}

		if isParentFileNotFoundDiag(diag) {
			l.Debug(
				"Filtering parent file not found diag",
				"diag", diag,
				"filename", filename,
			)

			continue
		}

		if isUnresolvableAttributeDiag(diag, text) {
			l.Debug(
				"Filtering unresolvable attribute diag",
				"diag", diag,
				"filename", filename,
			)

			continue
		}

		if diag.Subject.Filename == filename {
			filtered = append(filtered, diag)
		}
	}

	return filtered
}

const (
	// UnsupportedAttributeSummary is the summary for an unsupported attribute diagnostic.
	UnsupportedAttributeSummary = "Unsupported attribute"

	// OutputsMissingDetail is the detail for a missing outputs attribute diagnostic.
	OutputsMissingDetail = "This object does not have an attribute named \"outputs\"."
)

func isMissingOutputDiag(diag *hcl.Diagnostic) bool {
	if diag.Summary != UnsupportedAttributeSummary {
		return false
	}

	if filepath.Base(diag.Subject.Filename) == "terragrunt.hcl" {
		return false
	}

	return diag.Detail == OutputsMissingDetail
}

const (
	// ErrorInFunctionCallSummary is the summary for an error in a function call diagnostic.
	ErrorInFunctionCallSummary = "Error in function call"

	// ParentFileNotFoundErrorDetailPartial is the partial detail for a parent file not found diagnostic.
	ParentFileNotFoundErrorDetailPartial = `Call to function "find_in_parent_folders" failed: ParentFileNotFoundError`
)

func isParentFileNotFoundDiag(diag *hcl.Diagnostic) bool {
	if diag.Summary != ErrorInFunctionCallSummary {
		return false
	}

	return strings.HasPrefix(diag.Detail, ParentFileNotFoundErrorDetailPartial)
}

const (
	// UnknownVariableSummary is the summary for an unknown variable diagnostic.
	UnknownVariableSummary = "Unknown variable"

	// ValuesUnknownVariableDetail is the detail for a missing "values" variable diagnostic.
	ValuesUnknownVariableDetail = `There is no variable named "values".`
)

// unresolvableKeywords lists object names whose attributes may not be
// available during LS parsing because they depend on runtime state:
//   - "values": populated from terragrunt.values.hcl at runtime.
//   - "local": locals that reference unresolvable values cascade failures.
var unresolvableKeywords = []string{"values", "local"}

// isUnresolvableAttributeDiag checks whether the diagnostic is an "Unsupported
// attribute" or "Unknown variable" error caused by referencing an object that
// cannot be fully resolved during LS parsing.
func isUnresolvableAttributeDiag(diag *hcl.Diagnostic, text string) bool {
	if diag.Summary == UnknownVariableSummary && diag.Detail == ValuesUnknownVariableDetail {
		return true
	}

	if diag.Summary != UnsupportedAttributeSummary {
		return false
	}

	lines := strings.Split(text, "\n")
	line := diag.Subject.Start.Line - 1 // HCL lines are 1-based

	if line < 0 || line >= len(lines) {
		return false
	}

	// The diagnostic start column points to the "." in "keyword.attr".
	// Check that the characters immediately before the dot match a known keyword.
	col := diag.Subject.Start.Column - 1 // HCL columns are 1-based, convert to 0-based

	for _, keyword := range unresolvableKeywords {
		kLen := len(keyword)

		if col < kLen {
			continue
		}

		if lines[line][col-kLen:col] == keyword {
			return true
		}
	}

	return false
}

func hclDiagsToLSPDiags(hclDiags hcl.Diagnostics) []protocol.Diagnostic {
	diags := make([]protocol.Diagnostic, 0, len(hclDiags))

	for _, diag := range hclDiags {
		var diagRange protocol.Range

		if diag.Subject != nil {
			diagRange = protocol.Range{
				Start: protocol.Position{
					Line:      uint32(diag.Subject.Start.Line) - 1,
					Character: uint32(diag.Subject.Start.Column) - 1,
				},
				End: protocol.Position{
					Line:      uint32(diag.Subject.End.Line) - 1,
					Character: uint32(diag.Subject.End.Column) - 1,
				},
			}
		}

		diags = append(diags, protocol.Diagnostic{
			Range:    diagRange,
			Severity: protocol.DiagnosticSeverity(diag.Severity),
			Source:   "HCL",
			Message:  diag.Summary + ": " + diag.Detail,
		})
	}

	return diags
}

// DetectFileType returns the FileType for the given filename based on its base name.
func DetectFileType(filename string) store.FileType {
	base := filepath.Base(filename)

	switch base {
	case "terragrunt.hcl":
		return store.FileTypeUnit
	case "terragrunt.stack.hcl":
		return store.FileTypeStack
	case "terragrunt.values.hcl":
		return store.FileTypeValues
	default:
		return store.FileTypeUnknown
	}
}

// ParseStackBuffer parses a terragrunt.stack.hcl file and returns the stack config and diagnostics.
func ParseStackBuffer(ctx context.Context, l logger.Logger, filename, text string) (*config.StackConfig, []protocol.Diagnostic) {
	tgLogger := newTGLogger(l)
	ctx, s := newParsingSetup(ctx, tgLogger, filename)

	cfg, err := config.ReadStackConfigString(ctx, tgLogger, s.pctx, filename, text, nil)
	if err != nil {
		// Just log the error for now
		l.Error("Error parsing stack config", "error", err)
	}

	filteredDiags := filterHCLDiags(l, s.parseDiags, filename, text)

	diags := hclDiagsToLSPDiags(filteredDiags)

	return cfg, diags
}
