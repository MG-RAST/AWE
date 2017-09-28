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

type DummyRequirement struct {
	BaseRequirement `bson:",inline" yaml:",inline" json:",inline" mapstructure:",squash"`
}

func NewRequirement(class string, obj interface{}) (r Requirement, err error) {

	if class == "" {
		err = fmt.Errorf("class name empty")
		return
	}

	switch class {
	case "DockerRequirement":
		r, err = NewDockerRequirement(obj)
		if err != nil {
			fmt.Errorf("(NewRequirement) NewDockerRequirement returns: %s", err.Error())
			return
		}
		return
	case "InlineJavascriptRequirement":
		r, err = NewInlineJavascriptRequirement(obj)
		if err != nil {
			fmt.Errorf("(NewRequirement) NewInlineJavascriptRequirement returns: %s", err.Error())
			return
		}
		return
	case "EnvVarRequirement":
		r, err = NewEnvVarRequirement(obj)
		if err != nil {
			fmt.Errorf("(NewRequirement) NewEnvVarRequirement returns: %s", err.Error())
			return
		}
		return
	case "StepInputExpressionRequirement":
		r, err = NewStepInputExpressionRequirement(obj)
		if err != nil {
			fmt.Errorf("(NewRequirement) NewStepInputExpressionRequirement returns: %s", err.Error())
			return
		}
		return
	case "ShockRequirement":
		r, err = NewShockRequirement(obj)
		if err != nil {
			fmt.Errorf("(NewRequirement) NewShockRequirement returns: %s", err.Error())
			return
		}
		return
	case "InitialWorkDirRequirement":
		r, err = NewInitialWorkDirRequirement(obj)
		if err != nil {
			fmt.Errorf("(NewRequirement) NewInitialWorkDirRequirement returns: %s", err.Error())
			return
		}
		return
	case "ScatterFeatureRequirement":
		r, err = NewScatterFeatureRequirement(obj)
		if err != nil {
			fmt.Errorf("(NewRequirement) NewScatterFeatureRequirement returns: %s", err.Error())
			return
		}
		return
	case "MultipleInputFeatureRequirement":
		r, err = NewMultipleInputFeatureRequirement(obj)
		if err != nil {
			fmt.Errorf("(NewRequirement) NewMultipleInputFeatureRequirement returns: %s", err.Error())
			return
		}
		return
	case "SubworkflowFeatureRequirement":
		r = DummyRequirement{}
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

				err = fmt.Errorf("(CreateRequirementArray) A NewRequirement returns: %s", xerr)
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
				err = fmt.Errorf("(CreateRequirementArray) B NewRequirement returns: %s", xerr)
				return
			}

			new_array = append(new_array, requirement)
		}

	default:
		err = fmt.Errorf("(CreateRequirementArray) type unknown")
	}
	return
}
