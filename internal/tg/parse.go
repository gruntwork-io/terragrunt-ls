package tg

import (
	"context"
	"path/filepath"
	"strings"

	"terragrunt-ls/internal/logger"

	"github.com/gruntwork-io/terragrunt/config"
	"github.com/gruntwork-io/terragrunt/config/hclparse"
	"github.com/gruntwork-io/terragrunt/options"
	tgLog "github.com/gruntwork-io/terragrunt/pkg/log"
	"github.com/gruntwork-io/terragrunt/pkg/log/format"
	"github.com/gruntwork-io/terragrunt/tf"
	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"
	"github.com/zclconf/go-cty/cty"
	"go.lsp.dev/protocol"
)

// TerragruntFileType represents the type of Terragrunt configuration file
type TerragruntFileType int

const (
	// TerragruntFileTypeUnknown represents an unknown file type
	TerragruntFileTypeUnknown TerragruntFileType = iota
	// TerragruntFileTypeConfig represents a standard terragrunt.hcl file
	TerragruntFileTypeConfig
	// TerragruntFileTypeStack represents a terragrunt.stack.hcl file
	TerragruntFileTypeStack
	// TerragruntFileTypeValues represents a terragrunt.values.hcl file
	TerragruntFileTypeValues
)

// GetTerragruntFileType determines the type of Terragrunt file based on its name
func GetTerragruntFileType(filename string) TerragruntFileType {
	base := filepath.Base(filename)

	switch {
	case base == config.DefaultStackFile:
		return TerragruntFileTypeStack
	case base == "terragrunt.values.hcl": // TODO: Get this added as a constant in the config package.
		return TerragruntFileTypeValues
	case base != tf.TerraformLockFile && strings.HasSuffix(base, ".hcl"):
		return TerragruntFileTypeConfig
	default:
		return TerragruntFileTypeUnknown
	}
}

func ParseTerragruntConfigBuffer(l logger.Logger, filename, text string) (*config.TerragruntConfig, []protocol.Diagnostic) {
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

	tgLogger := tgLog.New(
		tgLog.WithOutput(l.Writer()),
		tgLog.WithLevel(tgLog.FromLogrusLevel(logrus.Level(l.Level()))),
		tgLog.WithFormatter(format.NewFormatter(format.NewJSONFormatPlaceholders())),
	)

	ctx := config.NewParsingContext(context.TODO(), tgLogger, opts)
	ctx.ParserOptions = parseOptions

	cfg, err := config.ParseConfigString(ctx, tgLogger, filename, text, nil)
	if err != nil {
		// Just log the error for now
		l.Error("Error parsing Terragrunt config", "error", err)
	}

	filteredDiags := filterHCLDiags(l, parseDiags, filename)

	diags := hclDiagsToLSPDiags(filteredDiags)

	return cfg, diags
}

func ParseTerragruntStackBuffer(l logger.Logger, filename, text string) (*config.StackConfig, []protocol.Diagnostic) {
	// Create Terragrunt options for parsing
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
				Source:   "Terragrunt",
			},
		}
	}

	opts.SkipOutput = true
	opts.NonInteractive = true

	// Create Terragrunt logger
	tgLogger := tgLog.New(
		tgLog.WithOutput(l.Writer()),
		tgLog.WithLevel(tgLog.FromLogrusLevel(logrus.Level(l.Level()))),
		tgLog.WithFormatter(format.NewFormatter(format.NewJSONFormatPlaceholders())),
	)

	// Try to read values from the directory if they exist (for stack parsing context)
	var values *cty.Value

	dir := filepath.Dir(filename)
	if valuesResult, valuesErr := config.ReadValues(context.Background(), tgLogger, opts, dir); valuesErr == nil {
		values = valuesResult
	}

	// Parse stack configuration using Terragrunt's native parser
	stackConfig, err := config.ReadStackConfigString(
		context.Background(),
		tgLogger,
		opts,
		filename,
		text,
		values,
	)

	if err != nil {
		// Convert error to diagnostic
		return nil, []protocol.Diagnostic{
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: 0, Character: 0},
				},
				Message:  err.Error(),
				Severity: protocol.DiagnosticSeverityError,
				Source:   "Terragrunt",
			},
		}
	}

	l.Debug("Successfully parsed stack config", "filename", filename, "stacks_count", len(stackConfig.Stacks), "units_count", len(stackConfig.Units))

	return stackConfig, []protocol.Diagnostic{}
}

func ParseTerragruntValuesBuffer(l logger.Logger, filename, text string) (*hclparse.File, []protocol.Diagnostic) {
	// Create Terragrunt options for parsing
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
				Source:   "Terragrunt",
			},
		}
	}

	opts.SkipOutput = true
	opts.NonInteractive = true

	// Create Terragrunt logger
	tgLogger := tgLog.New(
		tgLog.WithOutput(l.Writer()),
		tgLog.WithLevel(tgLog.FromLogrusLevel(logrus.Level(l.Level()))),
		tgLog.WithFormatter(format.NewFormatter(format.NewJSONFormatPlaceholders())),
	)

	// Parse values from string using Terragrunt's parsing logic (simplified)
	parser := config.NewParsingContext(context.Background(), tgLogger, opts)

	var parseDiags hcl.Diagnostics

	parseOptions := []hclparse.Option{
		hclparse.WithDiagnosticsHandler(func(file *hcl.File, hclDiags hcl.Diagnostics) (hcl.Diagnostics, error) {
			parseDiags = append(parseDiags, hclDiags...)
			return hclDiags, nil
		}),
	}
	parser.ParserOptions = parseOptions

	hclFile, err := hclparse.NewParser(parser.ParserOptions...).ParseFromString(text, filename)

	if err != nil {
		// Convert parsing errors to diagnostics
		return nil, []protocol.Diagnostic{
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: 0, Character: 0},
				},
				Message:  err.Error(),
				Severity: protocol.DiagnosticSeverityError,
				Source:   "Terragrunt",
			},
		}
	}

	// Basic validation - check if the file parses as valid HCL
	if hclFile == nil {
		return nil, []protocol.Diagnostic{
			{
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: 0, Character: 0},
				},
				Message:  "Failed to parse values file as valid HCL",
				Severity: protocol.DiagnosticSeverityError,
				Source:   "Terragrunt",
			},
		}
	}

	// Process any parsing diagnostics
	filteredDiags := filterHCLDiags(l, parseDiags, filename)
	diags := hclDiagsToLSPDiags(filteredDiags)

	// Successfully parsed values file
	l.Debug("Successfully parsed values config", "filename", filename)

	return hclFile, diags
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
