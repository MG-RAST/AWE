package cwl

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
)

//http://www.commonwl.org/v1.0/CommandLineTool.html#CommandOutputEnumSchema
type CommandOutputEnumSchema struct {
	Symbols       []string              `yaml:"symbols,omitempty" bson:"symbols,omitempty" json:"symbols,omitempty"`
	Type          string                `yaml:"type,omitempty" bson:"type,omitempty" json:"type,omitempty"` // must be enum
	Label         string                `yaml:"label,omitempty" bson:"label,omitempty" json:"label,omitempty"`
	OutputBinding *CommandOutputBinding `yaml:"outputbinding,omitempty" bson:"outputbinding,omitempty" json:"outputbinding,omitempty"`
}

func (c *CommandOutputEnumSchema) Is_CommandOutputParameterType() {}

func NewCommandOutputEnumSchema(v map[string]interface{}) (schema *CommandOutputEnumSchema, err error) {

	schema = &CommandOutputEnumSchema{}
	err = mapstructure.Decode(v, schema)
	if err != nil {
		err = fmt.Errorf("(NewCommandOutputEnumSchema) decode error: %s", err.Error())
		return
	}

	return
}
