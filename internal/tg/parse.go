package tg

import (
	"context"

	"github.com/gruntwork-io/terragrunt/config"
	"github.com/gruntwork-io/terragrunt/config/hclparse"
	"github.com/gruntwork-io/terragrunt/options"
	"github.com/hashicorp/hcl/v2"
	"go.lsp.dev/protocol"
)

func parseTerragruntBuffer(filename, text string) (*config.TerragruntConfig, []protocol.Diagnostic) {
	var parseDiags hcl.Diagnostics

	parseOptions := []hclparse.Option{
		hclparse.WithDiagnosticsHandler(func(file *hcl.File, hclDiags hcl.Diagnostics) (hcl.Diagnostics, error) {
			parseDiags = append(parseDiags, hclDiags...)
			return hclDiags, nil
		}),
	}

	opts := options.NewTerragruntOptions()
	opts.SkipOutput = true
	opts.NonInteractive = true
	opts.TerragruntConfigPath = filename

	ctx := config.NewParsingContext(context.TODO(), opts)
	ctx.ParserOptions = parseOptions

	cfg, _ := config.ParseConfigString(ctx, filename, text, nil)

	diags := hclDiagsToLSPDiags(parseDiags)

	return cfg, diags
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
			Message:  diag.Summary,
		})
	}

	return diags
}
