package cwl

import (
	"github.com/mitchellh/mapstructure"
)

type ShockRequirement struct {
	BaseRequirement `bson:",inline" yaml:",inline" json:",inline" `
	Shock_api_url   string `yaml:"shock_api_url,omitempty" bson:"shock_api_url,omitempty" json:"shock_api_url,omitempty" mapstructure:"shock_api_url,omitempty"`
}

func (s ShockRequirement) GetId() string { return "None" }

func NewShockRequirement(url string) (requirement_ptr *ShockRequirement, err error) {
	var requirement ShockRequirement
	requirement.Class = "ShockRequirement"
	requirement.Shock_api_url = url
	requirement_ptr = &requirement
	return
}

func NewShockRequirementFromInterface(original interface{}) (r *ShockRequirement, err error) {
	var requirement ShockRequirement
	r = &requirement
	err = mapstructure.Decode(original, &requirement)

	requirement.Class = "ShockRequirement"

	return
}
