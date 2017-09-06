package requirements

import (
	"github.com/mitchellh/mapstructure"
)

// Indicates that the workflow platform must support multiple inbound data links listed in the source field of WorkflowStepInput.
type MultipleInputFeatureRequirement struct {
	BaseRequirement `bson:",inline" yaml:",inline" json:",inline"`
}

func (c MultipleInputFeatureRequirement) GetId() string { return "None" }

func NewMultipleInputFeatureRequirement(original interface{}) (r *MultipleInputFeatureRequirement, err error) {
	var requirement MultipleInputFeatureRequirement
	r = &requirement
	err = mapstructure.Decode(original, &requirement)

	requirement.Class = "MultipleInputFeatureRequirement"

	return
}
