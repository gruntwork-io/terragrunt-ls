// Package stackutils provides shared utilities for working with Terragrunt stack configurations.
package stackutils

import (
	"path/filepath"
	"slices"

	"github.com/gruntwork-io/terragrunt/pkg/config"
)

func findUnit(stackCfg *config.StackConfig, unitName string) *config.Unit {
	if stackCfg == nil {
		return nil
	}

	idx := slices.IndexFunc(stackCfg.Units, func(u *config.Unit) bool {
		return u != nil && u.Name == unitName
	})
	if idx < 0 {
		return nil
	}

	return stackCfg.Units[idx]
}

func findStack(stackCfg *config.StackConfig, stackName string) *config.Stack {
	if stackCfg == nil {
		return nil
	}

	idx := slices.IndexFunc(stackCfg.Stacks, func(s *config.Stack) bool {
		return s != nil && s.Name == stackName
	})
	if idx < 0 {
		return nil
	}

	return stackCfg.Stacks[idx]
}

// LookupUnitPath looks up the path for a unit from the parsed StackConfig
func LookupUnitPath(stackCfg *config.StackConfig, unitName string) (string, bool) {
	unit := findUnit(stackCfg, unitName)
	if unit == nil {
		return "", false
	}

	return unit.Path, true
}

// LookupStackPath looks up the path for a stack from the parsed StackConfig
func LookupStackPath(stackCfg *config.StackConfig, stackName string) (string, bool) {
	stack := findStack(stackCfg, stackName)
	if stack == nil {
		return "", false
	}

	return stack.Path, true
}

// LookupUnitDir returns the directory a unit is generated into under dir,
// honoring no_dot_terragrunt_stack.
func LookupUnitDir(stackCfg *config.StackConfig, unitName, dir string) (string, bool) {
	unit := findUnit(stackCfg, unitName)
	if unit == nil {
		return "", false
	}

	return config.GetUnitDir(dir, unit), true
}

// LookupStackDir returns the directory a stack is generated into under dir,
// honoring no_dot_terragrunt_stack. Terragrunt only exports this for units.
func LookupStackDir(stackCfg *config.StackConfig, stackName, dir string) (string, bool) {
	stack := findStack(stackCfg, stackName)
	if stack == nil {
		return "", false
	}

	if stack.NoStack != nil && *stack.NoStack {
		return filepath.Join(dir, stack.Path), true
	}

	return filepath.Join(dir, config.StackDir, stack.Path), true
}
