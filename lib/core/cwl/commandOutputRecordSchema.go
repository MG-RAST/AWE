package cwl

import ()

type CommandOutputRecordSchema struct {
	Type   string                     `yaml:"type,omitempty" bson:"type,omitempty" json:"type,omitempty"` // Must be record
	Fields []CommandOutputRecordField `yaml:"fields,omitempty" bson:"fields,omitempty" json:"fields,omitempty"`
	Label  string                     `yaml:"label,omitempty" bson:"label,omitempty" json:"label,omitempty"`
}

func (c *CommandOutputRecordSchema) Is_CommandOutputParameterType() {}

type CommandOutputRecordField struct{}
