package cwl

import (
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

// http://www.commonwl.org/v1.0/CommandLineTool.html#EnvVarRequirement
type EnvVarRequirement struct {
	BaseRequirement `bson:",inline" yaml:",inline" json:",inline" mapstructure:",squash"`
	EnvDef          []EnvironmentDef `yaml:"envDef,omitempty" bson:"envDef,omitempty" json:"envDef,omitempty" mapstructure:"envDef,omitempty"`
}

func (c EnvVarRequirement) GetID() string { return "None" }

func NewEnvVarRequirement(original interface{}, context *WorkflowContext) (r *EnvVarRequirement, err error) {

	original, err = MakeStringMap(original, context)
	if err != nil {
		err = fmt.Errorf("(NewEnvVarRequirement) MakeStringMap returned: %s", err.Error())
		return
	}

	obj_map, ok := original.(map[string]interface{})

	if !ok {
		err = fmt.Errorf("(NewEnvVarRequirement) type is not a map[string]interface{} (got %s)", reflect.TypeOf(original))
		return
	}

	enfDev, has_enfDev := obj_map["envDef"]
	if has_enfDev {
		obj_map["envDef"], err = GetEnfDefArray(enfDev)
		if err != nil {
			err = fmt.Errorf("(NewEnvVarRequirement) GetEnfDefArray returned: %s", err.Error())
			return
		}

	} else {
		err = fmt.Errorf("(NewEnvVarRequirement) envDef field empty")
		return
	}

	var requirement EnvVarRequirement
	r = &requirement
	err = mapstructure.Decode(obj_map, &requirement)

	requirement.Class = "EnvVarRequirement"

	if requirement.EnvDef == nil {
		err = fmt.Errorf("(NewEnvVarRequirement) EnvDef empty")
		return
	}

	return
}

func (r *EnvVarRequirement) Evaluate(inputs interface{}, context *WorkflowContext) (err error) {
	for i, _ := range r.EnvDef {
		err = r.EnvDef[i].Evaluate(inputs, context)
		if err != nil {
			err = fmt.Errorf("(EnvVarRequirement/Evaluate) Evaluate returned: %s", err.Error())
			return
		}

	}
	return
}
