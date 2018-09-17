package cwl

import (
	"fmt"
	"reflect"

	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
)

// http://www.commonwl.org/v1.0/CommandLineTool.html#CommandInputArraySchema
type CommandInputArraySchema struct { // Items, Type , Label
	ArraySchema  `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // Type, Label
	InputBinding *CommandLineBinding                                                   `yaml:"inputBinding,omitempty" bson:"inputBinding,omitempty" json:"inputBinding,omitempty"`
}

func (c *CommandInputArraySchema) Type2String() string { return "CommandInputArraySchema" }
func (c *CommandInputArraySchema) GetId() string       { return "" }

func NewCommandInputArraySchema() (coas *CommandInputArraySchema) {

	coas = &CommandInputArraySchema{}
	coas.ArraySchema = *NewArraySchema()

	return
}

func NewCommandInputArraySchemaFromInterface(original interface{}, schemata []CWLType_Type, context *WorkflowContext) (coas *CommandInputArraySchema, err error) {

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	coas = NewCommandInputArraySchema()

	switch original.(type) {

	case map[string]interface{}:
		original_map, ok := original.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewCommandInputArraySchemaFromInterface) type error b")
			return
		}

		items, ok := original_map["items"]
		if ok {
			var items_type []CWLType_Type

			//fmt.Println("items: ")
			//spew.Dump(items)

			items_type, err = NewCWLType_TypeArray(items, schemata, "CommandInput", false, context)
			if err != nil {
				fmt.Println("a CommandInputArraySchema:")
				spew.Dump(original)
				err = fmt.Errorf("(NewCommandInputArraySchemaFromInterface) NewCWLType_TypeArray returns: %s", err.Error())
				return
			}
			//fmt.Println("items_type: ")
			//spew.Dump(items_type)
			original_map["items"] = items_type

		}

		err = mapstructure.Decode(original, coas)
		if err != nil {
			err = fmt.Errorf("(NewCommandInputArraySchemaFromInterface) %s", err.Error())
			return
		}
	default:
		err = fmt.Errorf("NewCommandInputArraySchemaFromInterface, unknown type %s", reflect.TypeOf(original))
	}
	return
}
