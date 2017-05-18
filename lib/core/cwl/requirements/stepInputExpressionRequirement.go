package requirements

import (
	"github.com/mitchellh/mapstructure"
)

type StepInputExpressionRequirement struct {
	//Class         string `yaml:"class"`
}

func (c StepInputExpressionRequirement) GetClass() string { return "StepInputExpressionRequirement" }
func (c StepInputExpressionRequirement) GetId() string    { return "None" }

func NewStepInputExpressionRequirement(original interface{}) (r *StepInputExpressionRequirement, err error) {
	var requirement StepInputExpressionRequirement
	r = &requirement
	err = mapstructure.Decode(original, &requirement)
	return
}
