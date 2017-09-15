package cwl

import (
	"errors"
	"fmt"
	requirements "github.com/MG-RAST/AWE/lib/core/cwl/requirements"
	cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	//"github.com/MG-RAST/AWE/lib/logger"
	//"github.com/davecgh/go-spew/spew"
	//"github.com/mitchellh/mapstructure"
)

type Requirement interface {
	GetClass() string
}

func NewRequirement(class cwl_types.CWLType_Type, obj interface{}) (r Requirement, err error) {
	switch string(class) {
	case "DockerRequirement":
		r, err = requirements.NewDockerRequirement(obj)
		return
	case "InlineJavascriptRequirement":
		r, err = requirements.NewInlineJavascriptRequirement(obj)
		return
	case "EnvVarRequirement":
		r, err = requirements.NewEnvVarRequirement(obj)
		return
	case "StepInputExpressionRequirement":
		r, err = requirements.NewStepInputExpressionRequirement(obj)
		return
	case "ShockRequirement":
		r, err = requirements.NewShockRequirement(obj)
		return
	case "InitialWorkDirRequirement":
		r, err = requirements.NewInitialWorkDirRequirement(obj)
		return
	case "ScatterFeatureRequirement":
		r, err = requirements.NewScatterFeatureRequirement(obj)
		return
	case "MultipleInputFeatureRequirement":
		r, err = requirements.NewMultipleInputFeatureRequirement(obj)
		return
	default:
		err = errors.New("Requirement class not supported " + string(class))

	}
	return
}

func CreateRequirementArray(original interface{}) (new_array_ptr *[]Requirement, err error) {
	// here the keynames are actually class names

	new_array := []Requirement{}
	new_array_ptr = &new_array

	switch original.(type) {
	case map[interface{}]interface{}:
		for k, v := range original.(map[interface{}]interface{}) {

			//var requirement Requirement
			class_str := k.(string)

			class := cwl_types.CWLType_Type(class_str)

			requirement, xerr := NewRequirement(class, v)
			if xerr != nil {
				err = xerr
				return
			}

			new_array = append(new_array, requirement)
		}
	case []interface{}:
		for _, v := range original.([]interface{}) {

			empty, xerr := cwl_types.NewEmpty(v)
			if xerr != nil {
				err = xerr
				return
			}
			class_str := empty.GetClass()
			class := cwl_types.CWLType_Type(class_str)

			requirement, xerr := NewRequirement(class, v)
			if xerr != nil {
				err = xerr
				return
			}

			new_array = append(new_array, requirement)
		}

	default:
		err = fmt.Errorf("(CreateRequirementArray) type unknown")
	}
	return
}
