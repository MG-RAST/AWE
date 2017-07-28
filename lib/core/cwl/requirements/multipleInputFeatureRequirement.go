package requirements

import (
	"github.com/mitchellh/mapstructure"
)

// Indicates that the workflow platform must support multiple inbound data links listed in the source field of WorkflowStepInput.
type MultipleInputFeatureRequirement struct {
	//Class         string `yaml:"class"`
}

func (c MultipleInputFeatureRequirement) GetClass() string { return "MultipleInputFeatureRequirement" }
func (c MultipleInputFeatureRequirement) GetId() string    { return "None" }

func NewMultipleInputFeatureRequirement(original interface{}) (r *MultipleInputFeatureRequirement, err error) {
	var requirement MultipleInputFeatureRequirement
	r = &requirement
	err = mapstructure.Decode(original, &requirement)
	return
}
