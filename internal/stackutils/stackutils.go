// Package stackutils finds where the blocks of a parsed Terragrunt stack configuration are
// generated.
package stackutils

import (
	"github.com/gruntwork-io/terragrunt/pkg/config"
)

// UnitDirs returns the directories that the enabled instances of the unit named name
// generate into under stackDir, in declaration order. A unit with an `expansion` block has
// one instance per element, all sharing the block's label. stackCfg is nil when the stack
// file failed to parse, which yields no directories.
func UnitDirs(stackCfg *config.StackConfig, name, stackDir string) []string {
	if stackCfg == nil {
		return nil
	}

	var dirs []string

	for _, unit := range stackCfg.Units {
		if unit.Name == name && unit.IsEnabled() {
			dirs = append(dirs, unit.GeneratedPath(stackDir))
		}
	}

	return dirs
}

// StackDirs returns the directories that the enabled instances of the stack named name
// generate into under stackDir, in declaration order. A stack with an `expansion` block
// has one instance per element, all sharing the block's label. stackCfg is nil when the
// stack file failed to parse, which yields no directories.
func StackDirs(stackCfg *config.StackConfig, name, stackDir string) []string {
	if stackCfg == nil {
		return nil
	}

	var dirs []string

	for _, stack := range stackCfg.Stacks {
		if stack.Name == name && stack.IsEnabled() {
			dirs = append(dirs, stack.GeneratedPath(stackDir))
		}
	}

	return dirs
}
