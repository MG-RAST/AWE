package cwl

import (
	"fmt"
	"reflect"
)

type InputArraySchema struct { // Items, Type , Label
	ArraySchema  `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // Type, Label
	InputBinding *CommandLineBinding                                                   `yaml:"inputBinding,omitempty" bson:"inputBinding,omitempty" json:"inputBinding,omitempty"`
}

//func (c *InputArraySchema) Is_CommandOutputParameterType() {}

func (c *InputArraySchema) Type2String() string { return "CommandOutputArraySchema" }
func (c *InputArraySchema) GetID() string       { return "" }

func NewInputArraySchema() (coas *InputArraySchema) {

	coas = &InputArraySchema{}
	coas.Type = CWLArray

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

		var as *ArraySchema
		as, err = NewArraySchemaFromMap(original_map, schemata, "Input", context)
		if err != nil {
			err = fmt.Errorf("(NewInputArraySchemaFromInterface) NewArraySchemaFromMap returned: %s", err.Error())
			return
		}

		coas = &InputArraySchema{}
		coas.ArraySchema = *as

		inputBinding, has_inputBinding := original_map["inputBinding"]
		if has_inputBinding {

			coas.InputBinding, err = NewCommandLineBinding(inputBinding, context)
			if err != nil {
				err = fmt.Errorf("(NewInputArraySchemaFromInterface) NewCommandOutputBinding returned: %s", err.Error())
				return
			}
		}

	default:
		err = fmt.Errorf("NewInputArraySchema, unknown type %s", reflect.TypeOf(original))
	}
	return
}
