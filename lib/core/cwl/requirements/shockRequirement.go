package requirements

import (
	"github.com/mitchellh/mapstructure"
)

type ShockRequirement struct {
	Host string `yaml:"host"`
}

func (s ShockRequirement) GetClass() string { return "ShockRequirement" }
func (s ShockRequirement) GetId() string    { return "None" }

func NewShockRequirement(original interface{}) (r *ShockRequirement, err error) {
	var requirement ShockRequirement
	r = &requirement
	err = mapstructure.Decode(original, &requirement)
	return
}
