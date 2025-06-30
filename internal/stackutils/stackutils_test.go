package stackutils_test

import (
	"terragrunt-ls/internal/stackutils"
	"testing"

	"github.com/gruntwork-io/terragrunt/config"
	"github.com/stretchr/testify/assert"
)

func TestLookupUnitPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		stackCfg     *config.StackConfig
		unitName     string
		expectedPath string
		expectedOk   bool
	}{
		{
			name:         "nil config",
			stackCfg:     nil,
			unitName:     "database",
			expectedPath: "",
			expectedOk:   false,
		},
		{
			name: "unit found",
			stackCfg: &config.StackConfig{
				Units: []*config.Unit{
					{Name: "database", Path: "db", Source: "./database"},
					{Name: "app", Path: "app", Source: "./app"},
				},
			},
			unitName:     "database",
			expectedPath: "db",
			expectedOk:   true,
		},
		{
			name: "unit not found",
			stackCfg: &config.StackConfig{
				Units: []*config.Unit{
					{Name: "database", Path: "db", Source: "./database"},
				},
			},
			unitName:     "missing",
			expectedPath: "",
			expectedOk:   false,
		},
		{
			name: "empty units",
			stackCfg: &config.StackConfig{
				Units: []*config.Unit{},
			},
			unitName:     "database",
			expectedPath: "",
			expectedOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path, ok := stackutils.LookupUnitPath(tt.stackCfg, tt.unitName)

			assert.Equal(t, tt.expectedOk, ok)
			assert.Equal(t, tt.expectedPath, path)
		})
	}
}

func TestLookupStackPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		stackCfg     *config.StackConfig
		stackName    string
		expectedPath string
		expectedOk   bool
	}{
		{
			name:         "nil config",
			stackCfg:     nil,
			stackName:    "nested",
			expectedPath: "",
			expectedOk:   false,
		},
		{
			name: "stack found",
			stackCfg: &config.StackConfig{
				Stacks: []*config.Stack{
					{Name: "nested", Path: "nested", Source: "./nested-stack"},
					{Name: "other", Path: "other", Source: "./other-stack"},
				},
			},
			stackName:    "nested",
			expectedPath: "nested",
			expectedOk:   true,
		},
		{
			name: "stack not found",
			stackCfg: &config.StackConfig{
				Stacks: []*config.Stack{
					{Name: "nested", Path: "nested", Source: "./nested-stack"},
				},
			},
			stackName:    "missing",
			expectedPath: "",
			expectedOk:   false,
		},
		{
			name: "empty stacks",
			stackCfg: &config.StackConfig{
				Stacks: []*config.Stack{},
			},
			stackName:    "nested",
			expectedPath: "",
			expectedOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path, ok := stackutils.LookupStackPath(tt.stackCfg, tt.stackName)

			assert.Equal(t, tt.expectedOk, ok)
			assert.Equal(t, tt.expectedPath, path)
		})
	}
}
