package tg

import (
	"context"
	"path/filepath"
	"strings"
	"terragrunt-ls/internal/logger"

	"github.com/gruntwork-io/terragrunt/config"
	"github.com/gruntwork-io/terragrunt/config/hclparse"
	"github.com/gruntwork-io/terragrunt/options"
	"github.com/hashicorp/hcl/v2"
	"go.lsp.dev/protocol"
)

func parseTerragruntBuffer(l logger.Logger, filename, text string) (*config.TerragruntConfig, []protocol.Diagnostic) {
	var parseDiags hcl.Diagnostics

	parseOptions := []hclparse.Option{
		hclparse.WithDiagnosticsHandler(func(file *hcl.File, hclDiags hcl.Diagnostics) (hcl.Diagnostics, error) {
			parseDiags = append(parseDiags, hclDiags...)
			return hclDiags, nil
		}),
	}

	opts, err := options.NewTerragruntOptionsWithConfigPath(filename)
	if err != nil {
		return nil, []protocol.Diagnostic{
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: 0, Character: 0},
				},
				Message:  err.Error(),
				Severity: protocol.DiagnosticSeverityError,
				Source:   "HCL",
			},
		}
	}

	opts.SkipOutput = true
	opts.NonInteractive = true

	ctx := config.NewParsingContext(context.TODO(), opts)
	ctx.ParserOptions = parseOptions

	cfg, _ := config.ParseConfigString(ctx, filename, text, nil)

	filteredDiags := filterHCLDiags(l, parseDiags, filename)

	diags := hclDiagsToLSPDiags(filteredDiags)

	return cfg, diags
}

func filterHCLDiags(l logger.Logger, diags hcl.Diagnostics, filename string) hcl.Diagnostics {
	filtered := hcl.Diagnostics{}

	for _, diag := range diags {
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

func hclDiagsToLSPDiags(hclDiags hcl.Diagnostics) []protocol.Diagnostic {
	diags := []protocol.Diagnostic{}

	for _, diag := range hclDiags {
		diags = append(diags, protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      uint32(diag.Subject.Start.Line) - 1,
					Character: uint32(diag.Subject.Start.Column) - 1,
				},
				End: protocol.Position{
					Line:      uint32(diag.Subject.End.Line) - 1,
					Character: uint32(diag.Subject.End.Column) - 1,
				},
			},
			Severity: protocol.DiagnosticSeverity(diag.Severity),
			Source:   "HCL",
			Message:  diag.Summary + ": " + diag.Detail,
		})
	}

	return diags
}
