package cwl

import (
	"fmt"
)

// https://www.commonwl.org/v1.0/Workflow.html#InputEnumSchema

type InputEnumSchema struct {
	EnumSchema `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // provides Symbols, Type, Label, Name
	//Name         string                                                                `yaml:"name,omitempty" json:"name,omitempty" bson:"name,omitempty" mapstructure:"name,omitempty"`
	InputBinding *CommandLineBinding `yaml:"inputBinding,omitempty" json:"inputBinding,omitempty" bson:"inputBinding,omitempty" mapstructure:"inputBinding,omitempty"`
}

func NewInputEnumSchemaFromInterface(original interface{}, context *WorkflowContext) (ies *InputEnumSchema, err error) {

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	ies = &InputEnumSchema{}

	switch original.(type) {

	case map[string]interface{}:

		ies.EnumSchema, err = NewEnumSchemaFromInterface(original, context)
		if err != nil {
			err = fmt.Errorf("(NewInputEnumSchemaFromInterface) NewEnumSchemaFromInterface returns: %s", err.Error())
			return
		}

		original_map, _ := original.(map[string]interface{})

		inputBinding, has_inputBinding := original_map["inputBinding"]

		if has_inputBinding {

			var clb *CommandLineBinding
			clb, err = NewCommandLineBinding(inputBinding, context)
			if err != nil {
				err = fmt.Errorf("(NewInputEnumSchemaFromInterface) NewCommandLineBinding returned: %s", err.Error())
				return
			}

			ies.InputBinding = clb
		}

		// name_if, has_name := original_map["name"]
		// if has_name {
		// 	name_str, ok := name_if.(string)
		// 	if !ok {
		// 		err = fmt.Errorf("(NewInputEnumSchemaFromInterface) field name is not string")
		// 		return
		// 	}
		// 	ies.Name = name_str
		// }

	default:
		err = fmt.Errorf("(NewInputEnumSchemaFromInterface) error")
		return

	}
	return
}
