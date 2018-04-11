package cwl

import (
	"github.com/mitchellh/mapstructure"
)

type InlineJavascriptRequirement struct {
	BaseRequirement `bson:",inline" yaml:",inline" json:",inline" mapstructure:",squash"`
	ExpressionLib   []string `yaml:"expressionLib,omitempty" bson:"expressionLib,omitempty" json:"expressionLib,omitempty"`
}

func (c InlineJavascriptRequirement) GetId() string { return "None" }

func NewInlineJavascriptRequirement() InlineJavascriptRequirement {
	var requirement InlineJavascriptRequirement
	requirement.Class = "InlineJavascriptRequirement"
	return requirement
}

func NewInlineJavascriptRequirementFromInterface(original interface{}) (r *InlineJavascriptRequirement, err error) {
	var requirement InlineJavascriptRequirement
	r = &requirement
	err = mapstructure.Decode(original, &requirement)

	requirement.Class = "InlineJavascriptRequirement"

	return
}
