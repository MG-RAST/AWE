package cwl

import (
	"errors"
	"fmt"
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
	fmt.Printf("CreateAnyArray :::::::::::::::::::")

	for k, v := range original.(map[interface{}]interface{}) {
		//fmt.Printf("A")

		switch v.(type) {
		case map[interface{}]interface{}: // the Requirement is a struct itself
			fmt.Printf("match")
			vmap := v.(map[interface{}]interface{})
			vmap["id"] = k.(string)
			requirement := k.(Requirement)
			new_array = append(new_array, requirement)
		default:
			fmt.Printf("not match")
			return errors.New("error"), nil
		}

	}
	//spew.Dump(new_array)
	return
}
