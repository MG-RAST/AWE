package cwl

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
	"reflect"
)

type OutputArraySchema struct { // Items, Type , Label
	ArraySchema   `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
	OutputBinding *CommandOutputBinding `yaml:"outputBinding,omitempty" bson:"outputBinding,omitempty" json:"outputBinding,omitempty"`
}

//func (c *CommandOutputArraySchema) Is_CommandOutputParameterType() {}

func (c *OutputArraySchema) Type2String() string { return "OutputArraySchema" }

func NewOutputArraySchema() (coas *OutputArraySchema) {

	coas = &OutputArraySchema{}
	coas.Type = "array"

	return
}

func NewOutputArraySchemaFromInterface(original interface{}) (coas *OutputArraySchema, err error) {

	original, err = MakeStringMap(original)
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
		if ok {
			var items_type []CWLType_Type
			items_type, err = NewCWLType_TypeArray(items, "Output")
			if err != nil {
				err = fmt.Errorf("(NewOutputArraySchema) NewCWLType_TypeArray returns: %s", err.Error())
				return
			}
			original_map["items"] = items_type

		}

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
