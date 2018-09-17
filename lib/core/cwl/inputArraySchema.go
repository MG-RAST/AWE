package cwl

import (
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

type InputArraySchema struct { // Items, Type , Label
	ArraySchema  `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // Type, Label
	InputBinding *CommandLineBinding                                                   `yaml:"inputBinding,omitempty" bson:"inputBinding,omitempty" json:"inputBinding,omitempty"`
}

//func (c *InputArraySchema) Is_CommandOutputParameterType() {}

func (c *InputArraySchema) Type2String() string { return "CommandOutputArraySchema" }
func (c *InputArraySchema) GetId() string       { return "" }

func NewInputArraySchema() (coas *InputArraySchema) {

	coas = &InputArraySchema{}
	coas.Type = CWL_array

	return
}

func NewInputArraySchemaFromInterface(original interface{}, schemata []CWLType_Type, context *WorkflowContext) (coas *InputArraySchema, err error) {

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	coas = NewInputArraySchema()

	switch original.(type) {

	case map[string]interface{}:
		original_map, ok := original.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewInputArraySchema) type error b")
			return
		}

		items, ok := original_map["items"]
		if ok {
			var items_type []CWLType_Type
			items_type, err = NewCWLType_TypeArray(items, schemata, "Input", false, context)
			if err != nil {
				err = fmt.Errorf("(NewInputArraySchema) NewCWLType_TypeArray returns: %s", err.Error())
				return
			}
			original_map["items"] = items_type

		}

		err = mapstructure.Decode(original, coas)
		if err != nil {
			err = fmt.Errorf("(NewCInputArraySchema) %s", err.Error())
			return
		}
	default:
		err = fmt.Errorf("NewInputArraySchema, unknown type %s", reflect.TypeOf(original))
	}
	return
}
