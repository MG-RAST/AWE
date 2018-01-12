package cwl

import (
	"github.com/mitchellh/mapstructure"
)

// http://www.commonwl.org/draft-3/CommandLineTool.html#ResourceRequirement

type ResourceRequirement struct {
	BaseRequirement `bson:",inline" yaml:",inline" json:",inline" mapstructure:",squash"`
	CoresMin        int `yaml:"coresMin,omitempty" bson:"coresMin,omitempty" json:"coresMin,omitempty"`
	CoresMax        int `yaml:"coresMax,omitempty" bson:"coresMax,omitempty" json:"coresMax,omitempty"`
	RamMin          int `yaml:"ramMin,omitempty" bson:"ramMin,omitempty" json:"ramMin,omitempty"`
	RamMax          int `yaml:"ramMax,omitempty" bson:"ramMax,omitempty" json:"ramMax,omitempty"`
	TmpdirMin       int `yaml:"tmpdirMin,omitempty" bson:"tmpdirMin,omitempty" json:"tmpdirMin,omitempty"`
	TmpdirMax       int `yaml:"tmpdirMax,omitempty" bson:"tmpdirMax,omitempty" json:"tmpdirMax,omitempty"`
	OutdirMin       int `yaml:"outdirMin,omitempty" bson:"outdirMin,omitempty" json:"outdirMin,omitempty"`
	OutdirMax       int `yaml:"outdirMax,omitempty" bson:"outdirMax,omitempty" json:"outdirMax,omitempty"`
}

func (c ResourceRequirement) GetId() string { return "None" }

func NewResourceRequirement(original interface{}) (r *ResourceRequirement, err error) {
	var requirement ResourceRequirement
	r = &requirement
	err = mapstructure.Decode(original, &requirement)

	requirement.Class = "ResourceRequirement"
	return
}
