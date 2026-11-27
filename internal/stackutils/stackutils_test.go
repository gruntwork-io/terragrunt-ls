package stackutils_test

import (
	"terragrunt-ls/internal/stackutils"
	"testing"

	"github.com/gruntwork-io/terragrunt/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestUnitDirs(t *testing.T) {
	t.Parallel()

	cfg := &config.StackConfig{
		Units: []*config.Unit{
			{Name: "generated", Path: "gen"},
			{Name: "explicit_false", Path: "ef", NoStack: new(false)},
			{Name: "in_place", Path: "ip", NoStack: new(true)},
			{Name: "expanded", Path: "app-a"},
			{Name: "expanded", Path: "app-b", Enabled: new(false)},
			{Name: "expanded", Path: "app-c"},
		},
	}

	tests := []struct {
		name     string
		block    string
		expected []string
	}{
		{name: "generated", block: "generated", expected: []string{"/s/.terragrunt-stack/gen"}},
		{name: "no_dot_terragrunt_stack false", block: "explicit_false", expected: []string{"/s/.terragrunt-stack/ef"}},
		{name: "no_dot_terragrunt_stack true", block: "in_place", expected: []string{"/s/ip"}},
		{
			name:     "enabled instances in declaration order",
			block:    "expanded",
			expected: []string{"/s/.terragrunt-stack/app-a", "/s/.terragrunt-stack/app-c"},
		},
		{name: "missing", block: "missing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, stackutils.UnitDirs(cfg, tt.block, "/s"))
		})
	}

	t.Run("nil config", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, stackutils.UnitDirs(nil, "generated", "/s"))
	})
}

func TestStackDirs(t *testing.T) {
	t.Parallel()

	cfg := &config.StackConfig{
		Stacks: []*config.Stack{
			{Name: "generated", Path: "gen"},
			{Name: "in_place", Path: "ip", NoStack: new(true)},
			{Name: "disabled", Path: "off", Enabled: new(false)},
			{Name: "expanded", Path: "env-a"},
			{Name: "expanded", Path: "env-b"},
		},
	}

	tests := []struct {
		name     string
		block    string
		expected []string
	}{
		{name: "generated", block: "generated", expected: []string{"/s/.terragrunt-stack/gen"}},
		{name: "no_dot_terragrunt_stack true", block: "in_place", expected: []string{"/s/ip"}},
		{name: "disabled", block: "disabled"},
		{
			name:     "instances in declaration order",
			block:    "expanded",
			expected: []string{"/s/.terragrunt-stack/env-a", "/s/.terragrunt-stack/env-b"},
		},
		{name: "missing", block: "missing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, stackutils.StackDirs(cfg, tt.block, "/s"))
		})
	}

	t.Run("nil config", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, stackutils.StackDirs(nil, "generated", "/s"))
	})
}
