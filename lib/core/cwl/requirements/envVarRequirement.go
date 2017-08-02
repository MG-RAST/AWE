package requirements

import (
	cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	"github.com/mitchellh/mapstructure"
)

type EnvVarRequirement struct {
	BaseRequirement `bson:",inline" yaml:",inline" json:",inline"`
	enfDef          []EnvironmentDef `yaml:"enfDef" bson:"enfDef" json:"enfDef"`
}

func (c EnvVarRequirement) GetId() string { return "None" }

func NewEnvVarRequirement(original interface{}) (r *EnvVarRequirement, err error) {
	var requirement EnvVarRequirement
	r = &requirement
	err = mapstructure.Decode(original, &requirement)

	requirement.Class = "EnvVarRequirement"

	return
}

type EnvironmentDef struct {
	envName  string               `yaml:"envName"`
	envValue cwl_types.Expression `yaml:"envValue"`
}
