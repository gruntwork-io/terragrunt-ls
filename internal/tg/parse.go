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

func newTGLogger(l logger.Logger) tgLog.Logger {
	return tgLog.New(
		tgLog.WithOutput(l.Writer()),
		tgLog.WithLevel(tgLog.FromLogrusLevel(logrus.Level(l.Level()))),
		tgLog.WithFormatter(format.NewFormatter(format.NewJSONFormatPlaceholders())),
	)
}

// newParsingContext builds a terragrunt ParsingContext for the given file and
// returns a pointer to the HCL diagnostics slice that the parser will populate.
func newParsingContext(ctx context.Context, tgLogger tgLog.Logger, filename string) (context.Context, *config.ParsingContext, *hcl.Diagnostics) {
	parseDiags := &hcl.Diagnostics{}

	parseOptions := []hclparse.Option{
		hclparse.WithDiagnosticsHandler(func(file *hcl.File, hclDiags hcl.Diagnostics) (hcl.Diagnostics, error) {
			*parseDiags = append(*parseDiags, hclDiags...)
			return hclDiags, nil
		}),
	}

	ctx, pctx := config.NewParsingContext(ctx, tgLogger)
	pctx.TerragruntConfigPath = filename
	pctx.WorkingDir = filepath.Dir(filename)
	pctx.SkipOutput = true
	pctx.MaxFoldersToCheck = defaultMaxFoldersToCheck
	pctx.Writers.Writer = io.Discard
	pctx.Writers.ErrWriter = io.Discard
	pctx.ParserOptions = append(pctx.ParserOptions, parseOptions...)

	return ctx, pctx, parseDiags
}

func ParseTerragruntBuffer(ctx context.Context, l logger.Logger, filename, text string) (*config.TerragruntConfig, []protocol.Diagnostic) {
	tgLogger := newTGLogger(l)
	ctx, pctx, parseDiags := newParsingContext(ctx, tgLogger, filename)

	cfg, err := config.ParseConfigString(ctx, pctx, tgLogger, filename, text, nil)
	if err != nil {
		// Just log the error for now
		l.Error("Error parsing Terragrunt config", "error", err)
	}

	filteredDiags := filterHCLDiags(l, *parseDiags, filename, text)

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

	// unknownVariableDetailPrefix is the start of an "Unknown variable" diagnostic
	// detail; the keyword name follows in quotes.
	unknownVariableDetailPrefix = `There is no variable named "`
)

// unresolvableKeywords lists object names whose attributes may not be
// available during LS parsing because they depend on runtime state:
//   - "values": populated from terragrunt.values.hcl at runtime.
//   - "local": locals that reference unresolvable values cascade failures.
//   - "include": child-config exposed-include attributes (e.g.,
//     `include.root.remote_state`) require resolving the parent file.
var unresolvableKeywords = []string{"values", "local", "include"}

// isUnresolvableAttributeDiag checks whether the diagnostic is an "Unsupported
// attribute" or "Unknown variable" error caused by referencing an object that
// cannot be fully resolved during LS parsing.
func isUnresolvableAttributeDiag(diag *hcl.Diagnostic, text string) bool {
	if diag.Summary == UnknownVariableSummary {
		for _, keyword := range unresolvableKeywords {
			if diag.Detail == unknownVariableDetailPrefix+keyword+`".` {
				return true
			}
		}
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
	ctx, pctx, parseDiags := newParsingContext(ctx, tgLogger, filename)

	cfg, err := config.ReadStackConfigString(ctx, tgLogger, pctx, filename, text, nil)
	if err != nil {
		// Just log the error for now
		l.Error("Error parsing stack config", "error", err)
	}

	filteredDiags := filterHCLDiags(l, *parseDiags, filename, text)

	diags := hclDiagsToLSPDiags(filteredDiags)

	return cfg, diags
}
