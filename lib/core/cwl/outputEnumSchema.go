package cwl

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

//http://www.commonwl.org/v1.0/Workflow.html#OutputEnumSchema
type OutputEnumSchema struct {
	EnumSchema    `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // provides Symbols, Type, Label
	OutputBinding *CommandOutputBinding                                                 `yaml:"outputbinding,omitempty" bson:"outputbinding,omitempty" json:"outputbinding,omitempty" mapstructure:"outputbinding,omitempty"`
}

func (c *OutputEnumSchema) Is_Type()            {}
func (c *OutputEnumSchema) Type2String() string { return "OutputEnumSchema" }
func (c *OutputEnumSchema) GetID() string       { return "" }

func NewOutputEnumSchemaFromInterface(v map[string]interface{}) (schema *OutputEnumSchema, err error) {

	schema = &OutputEnumSchema{}
	err = mapstructure.Decode(v, schema)
	if err != nil {
		err = fmt.Errorf("(NewOutputEnumSchemaFromInterface) decode error: %s", err.Error())
		return
	}

	return
}
