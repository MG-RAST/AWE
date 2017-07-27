package cwl

import (
	"errors"
	"fmt"
	requirements "github.com/MG-RAST/AWE/lib/core/cwl/requirements"
	cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	"github.com/MG-RAST/AWE/lib/logger"
	//"github.com/davecgh/go-spew/spew"
	//"github.com/mitchellh/mapstructure"
)

type Requirement interface {
	GetClass() string
}

func NewRequirement(class string, obj interface{}) (r Requirement, err error) {
	switch class {
	case "DockerRequirement":
		r, err = requirements.NewDockerRequirement(obj)
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
	default:
		err = errors.New("Requirement class not supported " + class)

	}
	return
}

// create Requirement object from interface type
func CreateRequirementDEPRECATED(class string, v interface{}) (requirement Requirement, err error) {
	switch v.(type) {
	case map[interface{}]interface{}: // the Requirement is a struct itself
		logger.Debug(1, "(CreateRequirementArray) type is map[interface{}]interface{}")
		vmap := v.(map[interface{}]interface{})

		vmap["class"] = class

		requirement, err = NewRequirement(class, v)
		if err != nil {
			return
		}

	case interface{}:
		logger.Debug(1, "(CreateRequirementArray) type is something else")
		requirement, err = NewRequirement(class, nil)
		if err != nil {
			return
		}
	default:
		err = fmt.Errorf("(CreateRequirement) type is unknown")

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
			class := k.(string)

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
			class := empty.GetClass()

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
