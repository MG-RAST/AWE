package cwl

import (
	"github.com/mitchellh/mapstructure"
)

//Indicates that the workflow platform must support the scatter and scatterMethod fields of WorkflowStep.
type ScatterFeatureRequirement struct {
	BaseRequirement `bson:",inline" yaml:",inline" json:",inline" mapstructure:",squash"`
}

func (c ScatterFeatureRequirement) GetID() string { return "None" }

func NewScatterFeatureRequirement(original interface{}) (r *ScatterFeatureRequirement, err error) {
	var requirement ScatterFeatureRequirement
	r = &requirement
	err = mapstructure.Decode(original, &requirement)

	requirement.Class = "ScatterFeatureRequirement"

	return
}
