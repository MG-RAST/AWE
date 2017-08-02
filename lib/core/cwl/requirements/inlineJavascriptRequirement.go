package requirements

import (
	"github.com/mitchellh/mapstructure"
)

type InlineJavascriptRequirement struct {
	BaseRequirement `bson:",inline" yaml:",inline" json:",inline"`
	ExpressionLib   []string `yaml:"expressionLib" bson:"expressionLib" json:"expressionLib"`
}

func (c InlineJavascriptRequirement) GetId() string { return "None" }

func NewInlineJavascriptRequirement(original interface{}) (r *InlineJavascriptRequirement, err error) {
	var requirement InlineJavascriptRequirement
	r = &requirement
	err = mapstructure.Decode(original, &requirement)

	requirement.Class = "InlineJavascriptRequirement"

	return
}
