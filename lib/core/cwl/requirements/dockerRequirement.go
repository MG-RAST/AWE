package requirements

import (
	"github.com/mitchellh/mapstructure"
)

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

func NewDockerRequirement(original interface{}) (r *DockerRequirement, err error) {
	var requirement DockerRequirement
	r = &requirement
	err = mapstructure.Decode(original, &requirement)
	return
}
