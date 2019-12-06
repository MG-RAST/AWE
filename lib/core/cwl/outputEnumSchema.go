package cwl

import (
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

// OutputEnumSchema http://www.commonwl.org/v1.0/Workflow.html#OutputEnumSchema
type OutputEnumSchema struct {
	EnumSchema    `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // provides Symbols, Type, Label
	OutputBinding *CommandOutputBinding                                                 `yaml:"outputbinding,omitempty" bson:"outputbinding,omitempty" json:"outputbinding,omitempty" mapstructure:"outputbinding,omitempty"`
}

// Is_Type _
func (c *OutputEnumSchema) Is_Type() {}

// Type2String _
func (c *OutputEnumSchema) Type2String() string { return "OutputEnumSchema" }

// GetID _
func (c *OutputEnumSchema) GetID() string { return "" }

// NewOutputEnumSchemaFromInterface _
func NewOutputEnumSchemaFromInterface(original interface{}, context *WorkflowContext) (schema *OutputEnumSchema, err error) {

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	switch original.(type) {

	case map[string]interface{}:

		originalMap, _ := original.(map[string]interface{})

		schema = &OutputEnumSchema{}
		err = mapstructure.Decode(originalMap, schema)
		if err != nil {
			err = fmt.Errorf("(NewOutputEnumSchemaFromInterface) decode error: %s", err.Error())
			return
		}
	default:
		err = fmt.Errorf("(NewOutputEnumSchemaFromInterface) type unknown: %s", reflect.TypeOf(original))
	}
	return
}
