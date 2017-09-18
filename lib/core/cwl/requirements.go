package cwl

import (
	"errors"
	"fmt"
	//"github.com/MG-RAST/AWE/lib/logger"
	//"github.com/davecgh/go-spew/spew"
	//"github.com/mitchellh/mapstructure"
)

type Requirement interface {
	GetClass() string
}

func NewRequirement(class string, obj interface{}) (r Requirement, err error) {
	switch class {
	case "DockerRequirement":
		r, err = NewDockerRequirement(obj)
		return
	case "InlineJavascriptRequirement":
		r, err = NewInlineJavascriptRequirement(obj)
		return
	case "EnvVarRequirement":
		r, err = NewEnvVarRequirement(obj)
		return
	case "StepInputExpressionRequirement":
		r, err = NewStepInputExpressionRequirement(obj)
		return
	case "ShockRequirement":
		r, err = NewShockRequirement(obj)
		return
	case "InitialWorkDirRequirement":
		r, err = NewInitialWorkDirRequirement(obj)
		return
	case "ScatterFeatureRequirement":
		r, err = NewScatterFeatureRequirement(obj)
		return
	case "MultipleInputFeatureRequirement":
		r, err = NewMultipleInputFeatureRequirement(obj)
		return
	default:
		err = errors.New("Requirement class not supported " + class)

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

			//class := CWLType_Type(class_str)

			requirement, xerr := NewRequirement(class_str, v)
			if xerr != nil {
				err = xerr
				return
			}

			new_array = append(new_array, requirement)
		}
	case []interface{}:
		for _, v := range original.([]interface{}) {

			empty, xerr := NewEmpty(v)
			if xerr != nil {
				err = xerr
				return
			}
			class_str := empty.GetClass()
			//class := CWLType_Type(class_str)

			requirement, xerr := NewRequirement(class_str, v)
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
