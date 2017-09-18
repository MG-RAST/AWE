package cwl

import (
	"github.com/mitchellh/mapstructure"
)

// http://www.commonwl.org/v1.0/CommandLineTool.html#InitialWorkDirRequirement
type InitialWorkDirRequirement struct {
	BaseRequirement `bson:",inline" yaml:",inline" json:",inline"`
	listing         string `yaml:"listing,omitempty" bson:"listing,omitempty" json:"listing,omitempty"` // TODO: array<File | Directory | Dirent | string | Expression> | string | Expression
}

func (c InitialWorkDirRequirement) GetId() string { return "None" }

func NewInitialWorkDirRequirement(original interface{}) (r *InitialWorkDirRequirement, err error) {
	var requirement InitialWorkDirRequirement
	r = &requirement
	err = mapstructure.Decode(original, &requirement)

	requirement.Class = "InlineJavascriptRequirement"

	return
}
