package requirements

import (
	"github.com/mitchellh/mapstructure"
)

//Indicates that the workflow platform must support the scatter and scatterMethod fields of WorkflowStep.
type ScatterFeatureRequirement struct {
	//Class         string `yaml:"class"`
}

func (c ScatterFeatureRequirement) GetClass() string { return "ScatterFeatureRequirement" }
func (c ScatterFeatureRequirement) GetId() string    { return "None" }

func NewScatterFeatureRequirement(original interface{}) (r *ScatterFeatureRequirement, err error) {
	var requirement ScatterFeatureRequirement
	r = &requirement
	err = mapstructure.Decode(original, &requirement)
	return
}
