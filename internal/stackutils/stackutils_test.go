package stackutils_test

import (
	"terragrunt-ls/internal/stackutils"
	"testing"

	"github.com/gruntwork-io/terragrunt/pkg/config"
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
			name: "nil entries are skipped",
			stackCfg: &config.StackConfig{
				Units: []*config.Unit{
					nil,
					{Name: "database", Path: "db", Source: "./database"},
				},
			},
			unitName:     "database",
			expectedPath: "db",
			expectedOk:   true,
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
			name: "nil entries are skipped",
			stackCfg: &config.StackConfig{
				Stacks: []*config.Stack{
					nil,
					{Name: "nested", Path: "nested", Source: "./nested-stack"},
				},
			},
			stackName:    "nested",
			expectedPath: "nested",
			expectedOk:   true,
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

func TestLookupDir(t *testing.T) {
	t.Parallel()

	yes := true
	no := false

	cfg := &config.StackConfig{
		Units: []*config.Unit{
			{Name: "generated", Path: "gen"},
			{Name: "explicit_false", Path: "ef", NoStack: &no},
			{Name: "in_place", Path: "ip", NoStack: &yes},
		},
		Stacks: []*config.Stack{
			{Name: "generated", Path: "gen"},
			{Name: "in_place", Path: "ip", NoStack: &yes},
		},
	}

	tests := []struct {
		lookup   func(*config.StackConfig, string, string) (string, bool)
		name     string
		block    string
		expected string
		found    bool
	}{
		{
			name:     "unit generated",
			lookup:   stackutils.LookupUnitDir,
			block:    "generated",
			expected: "/s/.terragrunt-stack/gen",
			found:    true,
		},
		{
			name:     "unit no_dot false",
			lookup:   stackutils.LookupUnitDir,
			block:    "explicit_false",
			expected: "/s/.terragrunt-stack/ef",
			found:    true,
		},
		{name: "unit no_dot true", lookup: stackutils.LookupUnitDir, block: "in_place", expected: "/s/ip", found: true},
		{name: "unit missing", lookup: stackutils.LookupUnitDir, block: "missing"},
		{
			name:     "stack generated",
			lookup:   stackutils.LookupStackDir,
			block:    "generated",
			expected: "/s/.terragrunt-stack/gen",
			found:    true,
		},
		{name: "stack no_dot true", lookup: stackutils.LookupStackDir, block: "in_place", expected: "/s/ip", found: true},
		{name: "stack missing", lookup: stackutils.LookupStackDir, block: "missing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir, ok := tt.lookup(cfg, tt.block, "/s")
			assert.Equal(t, tt.found, ok)
			assert.Equal(t, tt.expected, dir)
		})
	}

	t.Run("nil config", func(t *testing.T) {
		t.Parallel()

		_, ok := stackutils.LookupUnitDir(nil, "x", "/s")
		assert.False(t, ok)

		_, ok = stackutils.LookupStackDir(nil, "x", "/s")
		assert.False(t, ok)
	})
}
