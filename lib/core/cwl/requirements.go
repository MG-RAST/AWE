package cwl

import (
	"errors"
	"fmt"
	"reflect"
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

func NewRequirement(class string, obj interface{}) (r Requirement, schemata []CWLType_Type, err error) {

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
		r, schemata, err = NewSchemaDefRequirement(obj)
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

func CreateRequirementArray(original interface{}) (new_array_ptr *[]Requirement, schemata []CWLType_Type, err error) {
	// here the keynames are actually class names

	original, err = MakeStringMap(original)
	if err != nil {
		return
	}

	if original == nil {
		err = fmt.Errorf("(CreateRequirementArray) original == nil")
	}

	new_array := []Requirement{}

	switch original.(type) {
	case map[string]interface{}:
		for class_str, v := range original.(map[string]interface{}) {

			var schemata_new []CWLType_Type
			var requirement Requirement
			requirement, schemata_new, err = NewRequirement(class_str, v)
			if err != nil {
				err = fmt.Errorf("(CreateRequirementArray) A NewRequirement returns: %s", err)
				return
			}
			for i, _ := range schemata_new {
				schemata = append(schemata, schemata_new[i])
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
			var schemata_new []CWLType_Type
			var requirement Requirement
			requirement, schemata_new, err = NewRequirement(class_str, v)
			if err != nil {
				fmt.Println("CreateRequirementArray:")
				spew.Dump(original)
				fmt.Println("CreateRequirementArray done")
				err = fmt.Errorf("(CreateRequirementArray) B NewRequirement returns: %s (%s)", err, spew.Sdump(v))
				return
			}
			for i, _ := range schemata_new {
				schemata = append(schemata, schemata_new[i])
			}
			new_array = append(new_array, requirement)

		}

	default:
		err = fmt.Errorf("(CreateRequirementArray) type %s unknown", reflect.TypeOf(original))
	}

	new_array_ptr = &new_array

	return
}
