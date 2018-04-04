package cwl

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

type CommandOutputRecordSchema struct {
	Type   string                     `yaml:"type,omitempty" bson:"type,omitempty" json:"type,omitempty" mapstructure:"type,omitempty"` // Must be record
	Fields []CommandOutputRecordField `yaml:"fields,omitempty" bson:"fields,omitempty" json:"fields,omitempty" mapstructure:"fields,omitempty"`
	Label  string                     `yaml:"label,omitempty" bson:"label,omitempty" json:"label,omitempty" mapstructure:"label,omitempty"`
}

//func (c *CommandOutputRecordSchema) Is_CommandOutputParameterType() {}
func (c *CommandOutputRecordSchema) Is_Type()            {}
func (c *CommandOutputRecordSchema) Type2String() string { return "CommandOutputRecordSchema" }
func (c *CommandOutputRecordSchema) GetId() string       { return "" }

type CommandOutputRecordField struct{}

func NewCommandOutputRecordSchema(v interface{}) (schema *CommandOutputRecordSchema, err error) {

	schema = &CommandOutputRecordSchema{}
	err = mapstructure.Decode(v, schema)
	if err != nil {
		err = fmt.Errorf("(NewCommandOutputRecordSchema) decode error: %s", err.Error())
		return
	}

	return
}
