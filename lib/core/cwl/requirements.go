package cwl

import (
	"errors"
	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
)

type Requirement interface {
	Class() string
}

type DockerRequirement struct {
	//Class         string `yaml:"class"`
	DockerPull    string `yaml:"dockerPull"`
	DockerLoad    string `yaml:"dockerLoad"`
	DockerFile    string `yaml:"dockerFile"`
	DockerImport  string `yaml:"dockerImport"`
	DockerImageId string `yaml:"dockerImageId"`
}

func (c DockerRequirement) Class() string { return "DockerRequirement" }

// []Requirement
func CreateRequirementArray(original interface{}) (err error, new_array []Requirement) {
	// here the keynames are actually class names
	for k, v := range original.(map[interface{}]interface{}) {

		switch v.(type) {
		case map[interface{}]interface{}: // the Requirement is a struct itself
			vmap := v.(map[interface{}]interface{})
			class := k.(string)
			vmap["class"] = class

			switch {
			case class == "DockerRequirement":
				var requirement DockerRequirement
				err = mapstructure.Decode(v, &requirement)
				if err != nil {
					spew.Dump(v)
					return errors.New("object not a DockerRequirement"), nil

				}
				new_array = append(new_array, requirement)
			default:
				return errors.New("object class not supported " + class), nil

			}

		default:
			return errors.New("error: Requirement struct expected, but not struct found"), nil
		}

	}
	return
}
