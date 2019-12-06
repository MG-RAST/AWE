package cwl

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

//http://www.commonwl.org/v1.0/CommandLineTool.html#CommandOutputEnumSchema
type CommandOutputEnumSchema struct {
	EnumSchema    `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // provides Symbols, Type, Label
	OutputBinding *CommandOutputBinding                                                 `yaml:"outputbinding,omitempty" bson:"outputbinding,omitempty" json:"outputbinding,omitempty" mapstructure:"outputbinding,omitempty"`
}

//func (c *CommandOutputEnumSchema) Is_CommandOutputParameterType() {}
func (c *CommandOutputEnumSchema) Is_Type()            {}
func (c *CommandOutputEnumSchema) Type2String() string { return "CommandOutputEnumSchema" }
func (c *CommandOutputEnumSchema) GetID() string       { return "" }

func NewCommandOutputEnumSchema(v map[string]interface{}) (schema *CommandOutputEnumSchema, err error) {

	schema = &CommandOutputEnumSchema{}
	err = mapstructure.Decode(v, schema)
	if err != nil {
		err = fmt.Errorf("(NewCommandOutputEnumSchema) decode error: %s", err.Error())
		return
	}

	return
}
