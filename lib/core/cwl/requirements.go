package cwl

import (
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
)

type Requirement interface {
	GetClass() string
}

type StepInputExpressionRequirement struct {
	//Class         string `yaml:"class"`
}

func (c StepInputExpressionRequirement) GetClass() string { return "StepInputExpressionRequirement" }
func (c StepInputExpressionRequirement) GetId() string    { return "None" }

type DockerRequirement struct {
	//Class         string `yaml:"class"`
	DockerPull    string `yaml:"dockerPull"`
	DockerLoad    string `yaml:"dockerLoad"`
	DockerFile    string `yaml:"dockerFile"`
	DockerImport  string `yaml:"dockerImport"`
	DockerImageId string `yaml:"dockerImageId"`
}

func (c DockerRequirement) GetClass() string { return "DockerRequirement" }
func (c DockerRequirement) GetId() string    { return "None" }

type ShockRequirement struct {
	Host string `yaml:"host"`
}

func (s ShockRequirement) GetClass() string { return "ShockRequirement" }
func (s ShockRequirement) GetId() string    { return "None" }

func NewRequirement(class string, obj interface{}) (r Requirement, err error) {
	switch class {
	case "DockerRequirement":
		var requirement DockerRequirement
		err = mapstructure.Decode(obj, &requirement)
		if err != nil {
			spew.Dump(obj)
			err = fmt.Errorf("object not a DockerRequirement")
			return

		}
		r = requirement
	case "StepInputExpressionRequirement":
		var requirement StepInputExpressionRequirement
		r = requirement
	case "ShockRequirement":
		var requirement ShockRequirement
		err = mapstructure.Decode(obj, &requirement)
		if err != nil {
			spew.Dump(obj)
			err = fmt.Errorf("object not a DockerRequirement")
			return

		}
		r = requirement
	default:
		err = errors.New("object class not supported " + class)

	}
	return
}

// []Requirement
func CreateRequirementArray(original interface{}) (err error, new_array []Requirement) {
	// here the keynames are actually class names
	for k, v := range original.(map[interface{}]interface{}) {

		var requirement Requirement
		class := k.(string)

		switch v.(type) {
		case map[interface{}]interface{}: // the Requirement is a struct itself
			vmap := v.(map[interface{}]interface{})

			vmap["class"] = class

			requirement, err = NewRequirement(class, v)
			if err != nil {
				return
			}

		default:
			requirement, err = NewRequirement(class, nil)
			if err != nil {
				return
			}

		}

		new_array = append(new_array, requirement)
	}
	return
}