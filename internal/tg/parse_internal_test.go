package tg

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
)

func TestIsUnresolvableAttributeDiag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		diag *hcl.Diagnostic
		name string
		text string
		want bool
	}{
		{
			name: "values attribute access is unresolvable",
			diag: &hcl.Diagnostic{
				Summary: UnsupportedAttributeSummary,
				Subject: &hcl.Range{
					// Points at the "." in "values.environment".
					Start: hcl.Pos{Line: 2, Column: 23},
				},
			},
			text: "locals {\n  environment = values.environment\n}\n",
			want: true,
		},
		{
			name: "diagnostic column beyond buffer line must not panic",
			diag: &hcl.Diagnostic{
				Summary: UnsupportedAttributeSummary,
				Subject: &hcl.Range{
					// Position taken from a different (longer) file than the
					// parsed buffer, e.g. an included config.
					Start: hcl.Pos{Line: 1, Column: 29},
				},
			},
			text: "locals {}\n",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, isUnresolvableAttributeDiag(tt.diag, tt.text))
		})
	}
}
