package cwl

import (
	"fmt"
	"reflect"
)

// CommandInputArraySchema http://www.commonwl.org/v1.0/CommandLineTool.html#CommandInputArraySchema
type CommandInputArraySchema struct { // Items, Type , Label
	ArraySchema  `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // Type, Label, Items
	InputBinding *CommandLineBinding                                                   `yaml:"inputBinding,omitempty" bson:"inputBinding,omitempty" json:"inputBinding,omitempty"`
}

// Type2String _
func (c *CommandInputArraySchema) Type2String() string { return "CommandInputArraySchema" }

// GetID _
func (c *CommandInputArraySchema) GetID() string { return "" }

// NewCommandInputArraySchema _
func NewCommandInputArraySchema() (coas *CommandInputArraySchema) {

	coas = &CommandInputArraySchema{}
	coas.ArraySchema = *NewArraySchema()

	return
}

// NewCommandInputArraySchemaFromInterface _
func NewCommandInputArraySchemaFromInterface(original interface{}, schemata []CWLType_Type, context *WorkflowContext) (coas *CommandInputArraySchema, err error) {

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	switch original.(type) {

	case map[string]interface{}:
		originalMap, ok := original.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewCommandInputArraySchemaFromInterface) type error b")
			return
		}

		var as *ArraySchema
		as, err = NewArraySchemaFromMap(originalMap, schemata, "CommandInput", context)
		if err != nil {
			err = fmt.Errorf("(NewCommandInputArraySchemaFromInterface) NewArraySchemaFromMap returned: %s", err.Error())
			return
		}

		coas = &CommandInputArraySchema{}
		coas.ArraySchema = *as

		inputBinding, hasInputBinding := originalMap["inputBinding"]
		if hasInputBinding {

			coas.InputBinding, err = NewCommandLineBinding(inputBinding, context)
			if err != nil {
				err = fmt.Errorf("(NewOutputArraySchemaFromInterface) NewCommandOutputBinding returned: %s", err.Error())
				return
			}
		}

	default:
		err = fmt.Errorf("NewCommandInputArraySchemaFromInterface, unknown type %s", reflect.TypeOf(original))
	}
	return
}
