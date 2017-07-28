package requirements

import (
	"github.com/mitchellh/mapstructure"
)

type InlineJavascriptRequirement struct {
	//Class         string `yaml:"class"`
	ExpressionLib []string `yaml:"expressionLib"`
}

func (c InlineJavascriptRequirement) GetClass() string { return "InlineJavascriptRequirement" }
func (c InlineJavascriptRequirement) GetId() string    { return "None" }

func NewInlineJavascriptRequirement(original interface{}) (r *InlineJavascriptRequirement, err error) {
	var requirement InlineJavascriptRequirement
	r = &requirement
	err = mapstructure.Decode(original, &requirement)
	return
}
