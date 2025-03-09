package tg

import (
	"terragrunt-ls/internal/testutils"
	"testing"

	"github.com/gruntwork-io/terragrunt/config"
	"github.com/stretchr/testify/assert"
)

func TestParseTerragruntBuffer(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		content  string
		wantCfg  *config.TerragruntConfig
		wantDiag bool
	}{
		{
			name:     "basic terragrunt config",
			filename: "terragrunt.hcl",
			content: `
terraform {
  source = "./modules/example"
}
`,
			wantCfg:  &config.TerragruntConfig{},
			wantDiag: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := testutils.NewTestLogger(t)
			cfg, diags := ParseTerragruntBuffer(l, tt.filename, tt.content)

			if tt.wantDiag {
				assert.NotEmpty(t, diags, "expected diagnostics but got none")
			} else {
				assert.Empty(t, diags, "expected no diagnostics but got: %v", diags)
			}

			assert.NotNil(t, cfg, "expected config to not be nil")
		})
	}
}
