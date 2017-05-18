package requirements

import (
	cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	"github.com/mitchellh/mapstructure"
)

type EnvVarRequirement struct {
	//Class         string `yaml:"class"`
	enfDef []EnvironmentDef `yaml:"enfDef"`
}

func (c EnvVarRequirement) GetClass() string { return "EnvVarRequirement" }
func (c EnvVarRequirement) GetId() string    { return "None" }

func NewEnvVarRequirement(original interface{}) (r *EnvVarRequirement, err error) {
	var requirement EnvVarRequirement
	r = &requirement
	err = mapstructure.Decode(original, &requirement)
	return
}

type EnvironmentDef struct {
	envName  string               `yaml:"envName"`
	envValue cwl_types.Expression `yaml:"envValue"`
}
