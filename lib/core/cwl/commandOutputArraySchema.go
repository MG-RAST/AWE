package cwl

import (
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

// https://www.commonwl.org/draft-3/CommandLineTool.html#CommandOutputArraySchema
type CommandOutputArraySchema struct { // Items, Type , Label
	ArraySchema   `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` //provides Type, Label, Items
	OutputBinding *CommandOutputBinding                                                 `yaml:"outputBinding,omitempty" bson:"outputBinding,omitempty" json:"outputBinding,omitempty" mapstructure:"outputBinding,omitempty"`
}

//func (c *CommandOutputArraySchema) Is_CommandOutputParameterType() {}

func (c *CommandOutputArraySchema) Type2String() string { return "CommandOutputArraySchema" }
func (c *CommandOutputArraySchema) GetID() string       { return "" }

func NewCommandOutputArraySchema() (coas *CommandOutputArraySchema) {

	coas = &CommandOutputArraySchema{}
	coas.ArraySchema = *NewArraySchema()
	//coas.Type = CWLArray

	return
}

func NewCommandOutputArraySchemaFromInterface(original interface{}, schemata []CWLType_Type, context *WorkflowContext) (coas *CommandOutputArraySchema, err error) {

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	coas = NewCommandOutputArraySchema()

	switch original.(type) {

	case map[string]interface{}:
		original_map, ok := original.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewCommandOutputArraySchemaFromInterface) type error b")
			return
		}

		items, ok := original_map["items"]
		if ok {
			var items_type []CWLType_Type
			items_type, err = NewCWLType_TypeArray(items, schemata, "CommandOutput", false, context)
			if err != nil {
				err = fmt.Errorf("(NewCommandOutputArraySchemaFromInterface) NewCWLType_TypeArray returns: %s", err.Error())
				return
			}
			original_map["items"] = items_type

		}

		original_map["type"] = CWLArray

		err = mapstructure.Decode(original, coas)
		if err != nil {
			err = fmt.Errorf("(NewCommandOutputArraySchemaFromInterface) %s", err.Error())
			return
		}
		if coas.Type == nil {
			panic("nononono")
		}
	default:
		err = fmt.Errorf("NewCommandOutputArraySchemaFromInterface, unknown type %s", reflect.TypeOf(original))
	}
	return
}
