// Package stackutils provides shared utilities for working with Terragrunt stack configurations.
package stackutils

import (
	"github.com/gruntwork-io/terragrunt/config"
)

// LookupUnitPath looks up the path for a unit from the parsed StackConfig
func LookupUnitPath(stackCfg *config.StackConfig, unitName string) (string, bool) {
	if stackCfg == nil {
		return "", false
	}

	for _, unit := range stackCfg.Units {
		if unit != nil && unit.Name == unitName {
			return unit.Path, true
		}
	}

	return "", false
}

// LookupStackPath looks up the path for a stack from the parsed StackConfig
func LookupStackPath(stackCfg *config.StackConfig, stackName string) (string, bool) {
	if stackCfg == nil {
		return "", false
	}

	for _, stack := range stackCfg.Stacks {
		if stack != nil && stack.Name == stackName {
			return stack.Path, true
		}
	}

	return "", false
}
