package requirements

import (
	"github.com/mitchellh/mapstructure"
)

// http://www.commonwl.org/v1.0/CommandLineTool.html#InitialWorkDirRequirement
type InitialWorkDirRequirement struct {
	//Class         string `yaml:"class"`
	listing string `yaml:"listing"` // TODO: array<File | Directory | Dirent | string | Expression> | string | Expression
}

func (c InitialWorkDirRequirement) GetClass() string { return "InitialWorkDirRequirement" }
func (c InitialWorkDirRequirement) GetId() string    { return "None" }

func NewInitialWorkDirRequirement(original interface{}) (r *InitialWorkDirRequirement, err error) {
	var requirement InitialWorkDirRequirement
	r = &requirement
	err = mapstructure.Decode(original, &requirement)
	return
}
