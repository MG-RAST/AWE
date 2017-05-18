package requirements

import (
	"github.com/mitchellh/mapstructure"
)

type InitialWorkDirRequirement struct {
	//Class         string `yaml:"class"`
}

func (c InitialWorkDirRequirement) GetClass() string { return "InitialWorkDirRequirement" }
func (c InitialWorkDirRequirement) GetId() string    { return "None" }

func NewInitialWorkDirRequirement(original interface{}) (r *InitialWorkDirRequirement, err error) {
	var requirement InitialWorkDirRequirement
	r = &requirement
	err = mapstructure.Decode(original, &requirement)
	return
}
