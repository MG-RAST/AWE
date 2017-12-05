package cwl

import (
	"errors"
	"fmt"
	//"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
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
	case "ResourceRequirement":
		r, err = NewResourceRequirement(obj)
		if err != nil {
			fmt.Errorf("(NewRequirement) NewResourceRequirement returns: %s", err.Error())
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
	case "SchemaDefRequirement":
		r, err = NewSchemaDefRequirement(obj)
		if err != nil {
			fmt.Errorf("(NewRequirement) NewSchemaDefRequirement returns: %s", err.Error())
			return
		}
		return

	case "SubworkflowFeatureRequirement":
		this_r := DummyRequirement{}
		this_r.Class = "SubworkflowFeatureRequirement"
		r = this_r
	default:
		err = errors.New("Requirement class not supported " + class)

	}
	return
}

func CreateRequirementArray(original interface{}) (new_array_ptr *[]Requirement, err error) {
	// here the keynames are actually class names

	original, err = MakeStringMap(original)
	if err != nil {
		return
	}

	new_array := []Requirement{}

	switch original.(type) {
	case map[string]interface{}:
		for class_str, v := range original.(map[string]interface{}) {

			requirement, xerr := NewRequirement(class_str, v)
			if xerr != nil {

				err = fmt.Errorf("(CreateRequirementArray) A NewRequirement returns: %s", xerr)
				return
			}

			new_array = append(new_array, requirement)
		}
	case []interface{}:
		original_array := original.([]interface{})

		for i, _ := range original_array {
			v := original_array[i]

			var class_str string
			class_str, err = GetClass(v)
			if err != nil {
				return
			}

			//class := CWLType_Type(class_str)

			requirement, xerr := NewRequirement(class_str, v)
			if xerr != nil {
				fmt.Println("CreateRequirementArray:")
				spew.Dump(original)
				fmt.Println("CreateRequirementArray done")
				err = fmt.Errorf("(CreateRequirementArray) B NewRequirement returns: %s (%s)", xerr, spew.Sdump(v))
				return
			}

			new_array = append(new_array, requirement)

		}

	default:
		err = fmt.Errorf("(CreateRequirementArray) type unknown")
	}

	new_array_ptr = &new_array

	return
}
