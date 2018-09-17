package cwl

import (
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

type OutputArraySchema struct { // Items, Type , Label
	ArraySchema   `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
	OutputBinding *CommandOutputBinding `yaml:"outputBinding,omitempty" bson:"outputBinding,omitempty" json:"outputBinding,omitempty"`
}

//func (c *CommandOutputArraySchema) Is_CommandOutputParameterType() {}

func (c OutputArraySchema) Type2String() string { return "OutputArraySchema" }
func (c OutputArraySchema) GetId() string       { return "" }
func (c OutputArraySchema) Is_Type()            {}

func NewOutputArraySchema() (coas *OutputArraySchema) {

	coas = &OutputArraySchema{}
	coas.Type = CWL_array

	return
}

func NewOutputArraySchemaFromInterface(original interface{}, schemata []CWLType_Type, context *WorkflowContext) (coas *OutputArraySchema, err error) {

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	coas = NewOutputArraySchema()

	switch original.(type) {

	case map[string]interface{}:
		original_map, ok := original.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewOutputArraySchema) type error b")
			return
		}

		items, ok := original_map["items"]
		if !ok {

			err = fmt.Errorf("(NewOutputArraySchema) items are missing")
			return
		}
		var items_type []CWLType_Type
		items_type, err = NewCWLType_TypeArray(items, schemata, "Output", false, context)
		if err != nil {
			err = fmt.Errorf("(NewOutputArraySchema) NewCWLType_TypeArray returns: %s", err.Error())
			return
		}
		original_map["items"] = items_type

		err = mapstructure.Decode(original, coas)
		if err != nil {
			err = fmt.Errorf("(NewOutputArraySchema) %s", err.Error())
			return
		}
	default:
		err = fmt.Errorf("NewOutputArraySchema, unknown type %s", reflect.TypeOf(original))
	}
	return
}
