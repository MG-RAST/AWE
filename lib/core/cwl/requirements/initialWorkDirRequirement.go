package requirements

import (
	"github.com/mitchellh/mapstructure"
)

// http://www.commonwl.org/v1.0/CommandLineTool.html#InitialWorkDirRequirement
type InitialWorkDirRequirement struct {
	BaseRequirement `bson:",inline" yaml:",inline" json:",inline"`
	listing         string `yaml:"listing" bson:"listing" json:"listing"` // TODO: array<File | Directory | Dirent | string | Expression> | string | Expression
}

func (c InitialWorkDirRequirement) GetId() string { return "None" }

func NewInitialWorkDirRequirement(original interface{}) (r *InitialWorkDirRequirement, err error) {
	var requirement InitialWorkDirRequirement
	r = &requirement
	err = mapstructure.Decode(original, &requirement)

	requirement.Class = "InlineJavascriptRequirement"

	return
}
