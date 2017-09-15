package requirements

import (
	"github.com/mitchellh/mapstructure"
)

type DockerRequirement struct {
	BaseRequirement `bson:",inline" yaml:",inline" json:",inline"`
	DockerPull      string `yaml:"dockerPull,omitempty" bson:"dockerPull,omitempty" json:"dockerPull,omitempty"`
	DockerLoad      string `yaml:"dockerLoad,omitempty" bson:"dockerLoad,omitempty" json:"dockerLoad,omitempty"`
	DockerFile      string `yaml:"dockerFile,omitempty" bson:"dockerFile,omitempty" json:"dockerFile,omitempty"`
	DockerImport    string `yaml:"dockerImport,omitempty" bson:"dockerImport,omitempty" json:"dockerImport,omitempty"`
	DockerImageId   string `yaml:"dockerImageId,omitempty" bson:"dockerImageId,omitempty" json:"dockerImageId,omitempty"`
}

func (c DockerRequirement) GetId() string { return "None" }

func NewDockerRequirement(original interface{}) (r *DockerRequirement, err error) {
	var requirement DockerRequirement
	r = &requirement
	err = mapstructure.Decode(original, &requirement)

	requirement.Class = "DockerRequirement"
	return
}
