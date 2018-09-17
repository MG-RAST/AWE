package cwl

import (
	"fmt"
)

type InputEnumSchema struct {
	EnumSchema   `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // provides Symbols, Type, Label
	InputBinding *CommandLineBinding                                                   `yaml:"inputBinding,omitempty" json:"inputBinding,omitempty" bson:"inputBinding,omitempty" mapstructure:"inputBinding,omitempty"`
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
				err = fmt.Errorf("(NewInputEnumSchemaFromInterface) ")
			}

			ies.InputBinding = clb
		}

	default:
		err = fmt.Errorf("(NewInputEnumSchemaFromInterface) error")
		return

	}
	return
}
