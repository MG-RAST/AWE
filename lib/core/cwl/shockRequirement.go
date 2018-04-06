package cwl

import (
	"github.com/mitchellh/mapstructure"
)

type ShockRequirement struct {
	BaseRequirement `bson:",inline" yaml:",inline" json:",inline"`
	Host            string `yaml:"host,omitempty" bson:"host,omitempty" json:"host,omitempty"`
}

func (s ShockRequirement) GetId() string { return "None" }

func NewShockRequirement(host string) (r *ShockRequirement, err error) {
	var requirement ShockRequirement
	requirement.Class = "ShockRequirement"
	return
}

func NewShockRequirementFromInterface(original interface{}) (r *ShockRequirement, err error) {
	var requirement ShockRequirement
	r = &requirement
	err = mapstructure.Decode(original, &requirement)

	requirement.Class = "ShockRequirement"

	return
}
