package tg

import (
	"context"

	"github.com/gruntwork-io/terragrunt/config"
	"github.com/gruntwork-io/terragrunt/config/hclparse"
	"github.com/gruntwork-io/terragrunt/options"
	"github.com/hashicorp/hcl/v2"
)

func parseTerragruntBuffer(filename, text string) (*config.TerragruntConfig, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	parseOptions := []hclparse.Option{
		hclparse.WithDiagnosticsHandler(func(file *hcl.File, hclDiags hcl.Diagnostics) (hcl.Diagnostics, error) {
			diags = append(diags, hclDiags...)
			return diags, nil
		}),
	}

	opts := options.NewTerragruntOptions()
	opts.SkipOutput = true
	opts.NonInteractive = true

	ctx := config.NewParsingContext(context.TODO(), opts)
	ctx.ParserOptions = parseOptions

	cfg, _ := config.ParseConfigString(ctx, filename, text, nil)
	return cfg, diags
}
