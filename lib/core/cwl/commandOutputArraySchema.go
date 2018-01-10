package cwl

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
	"reflect"
)

type CommandOutputArraySchema struct { // Items, Type , Label
	ArraySchema   `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
	OutputBinding *CommandOutputBinding `yaml:"outputBinding,omitempty" bson:"outputBinding,omitempty" json:"outputBinding,omitempty"`
}

//func (c *CommandOutputArraySchema) Is_CommandOutputParameterType() {}

func (c *CommandOutputArraySchema) Type2String() string { return "CommandOutputArraySchema" }
func (c *CommandOutputArraySchema) GetId() string       { return "" }

func NewCommandOutputArraySchema() (coas *CommandOutputArraySchema) {

	coas = &CommandOutputArraySchema{}
	coas.Type = "array"

	return
}

func NewCommandOutputArraySchemaFromInterface(original interface{}, schemata []CWLType_Type) (coas *CommandOutputArraySchema, err error) {

	original, err = MakeStringMap(original)
	if err != nil {
		return
	}

	coas = NewCommandOutputArraySchema()

	switch original.(type) {

	case map[string]interface{}:
		original_map, ok := original.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewCommandOutputArraySchema) type error b")
			return
		}

		items, ok := original_map["items"]
		if ok {
			var items_type []CWLType_Type
			items_type, err = NewCWLType_TypeArray(items, schemata, "CommandOutput", false)
			if err != nil {
				err = fmt.Errorf("(NewCommandOutputArraySchema) NewCWLType_TypeArray returns: %s", err.Error())
				return
			}
			original_map["items"] = items_type

		}

		err = mapstructure.Decode(original, coas)
		if err != nil {
			err = fmt.Errorf("(NewCommandOutputArraySchema) %s", err.Error())
			return
		}
		if coas.Type == "" {
			panic("nononono")
		}
	default:
		err = fmt.Errorf("NewCommandOutputArraySchema, unknown type %s", reflect.TypeOf(original))
	}
	return
}
