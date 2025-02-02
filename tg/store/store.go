package store

import (
	"github.com/gruntwork-io/terragrunt/config"
	"github.com/zclconf/go-cty/cty"
)

type Store struct {
	Cfg      *config.TerragruntConfig
	CfgAsCty cty.Value
	Document string
}
