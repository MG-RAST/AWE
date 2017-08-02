package requirements

import (
	"github.com/mitchellh/mapstructure"
)

type DockerRequirement struct {
	BaseRequirement `bson:",inline" yaml:",inline" json:",inline"`
	DockerPull      string `yaml:"dockerPull" bson:"dockerPull" json:"dockerPull"`
	DockerLoad      string `yaml:"dockerLoad" bson:"dockerLoad" json:"dockerLoad"`
	DockerFile      string `yaml:"dockerFile" bson:"dockerFile" json:"dockerFile"`
	DockerImport    string `yaml:"dockerImport" bson:"dockerImport" json:"dockerImport"`
	DockerImageId   string `yaml:"dockerImageId" bson:"dockerImageId" json:"dockerImageId"`
}

func (c DockerRequirement) GetId() string { return "None" }

func NewDockerRequirement(original interface{}) (r *DockerRequirement, err error) {
	var requirement DockerRequirement
	r = &requirement
	err = mapstructure.Decode(original, &requirement)

	requirement.Class = "DockerRequirement"
	return
}
