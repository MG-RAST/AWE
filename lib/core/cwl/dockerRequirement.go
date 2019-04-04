package cwl

import (
	"github.com/mitchellh/mapstructure"
)

type DockerRequirement struct {
	BaseRequirement       `bson:",inline" yaml:",inline" json:",inline" mapstructure:",squash"`
	DockerPull            string `yaml:"dockerPull,omitempty" bson:"dockerPull,omitempty" json:"dockerPull,omitempty" mapstructure:"dockerPull,omitempty"`
	DockerLoad            string `yaml:"dockerLoad,omitempty" bson:"dockerLoad,omitempty" json:"dockerLoad,omitempty" mapstructure:"dockerLoad,omitempty"`
	DockerFile            string `yaml:"dockerFile,omitempty" bson:"dockerFile,omitempty" json:"dockerFile,omitempty" mapstructure:"dockerFile,omitempty"`
	DockerImport          string `yaml:"dockerImport,omitempty" bson:"dockerImport,omitempty" json:"dockerImport,omitempty" mapstructure:"dockerImport,omitempty"`
	DockerImageId         string `yaml:"dockerImageId,omitempty" bson:"dockerImageId,omitempty" json:"dockerImageId,omitempty" mapstructure:"dockerImageId,omitempty"`
	DockerOutputDirectory string `yaml:"dockerOutputDirectory,omitempty" bson:"dockerOutputDirectory,omitempty" json:"dockerOutputDirectory,omitempty" mapstructure:"dockerOutputDirectory,omitempty"`
}

func (c DockerRequirement) GetID() string { return "None" }

func NewDockerRequirement(original interface{}) (r *DockerRequirement, err error) {
	var requirement DockerRequirement
	r = &requirement
	err = mapstructure.Decode(original, &requirement)

	requirement.Class = "DockerRequirement"
	return
}
