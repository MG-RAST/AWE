package cwl

import (
	"github.com/mitchellh/mapstructure"
)

//https://www.commonwl.org/v1.0/CommandLineTool.html#ShellCommandRequirement
type ShellCommandRequirement struct {
	BaseRequirement `bson:",inline" yaml:",inline" json:",inline" mapstructure:",squash"`
}

func (c ShellCommandRequirement) GetId() string { return "None" }

func NewShellCommandRequirement(original interface{}) (r *ShellCommandRequirement, err error) {

	var requirement ShellCommandRequirement
	r = &requirement
	err = mapstructure.Decode(original, &requirement)

	requirement.Class = "ShellCommandRequirement"

	return
}
