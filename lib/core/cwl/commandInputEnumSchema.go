package cwl

import (
	"fmt"
)

// https://www.commonwl.org/v1.1.0-dev1/CommandLineTool.html#CommandInputEnumSchema
type CommandInputEnumSchema struct {
	EnumSchema   `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // provides Symbols, Type, Label
	Name         string                                                                `yaml:"name,omitempty" json:"name,omitempty" bson:"name,omitempty" mapstructure:"name,omitempty"`
	InputBinding *CommandLineBinding                                                   `yaml:"inputBinding,omitempty" json:"inputBinding,omitempty" bson:"inputBinding,omitempty" mapstructure:"inputBinding,omitempty"`
}

func NewCommandInputEnumSchemaFromInterface(original interface{}, context *WorkflowContext) (cies *CommandInputEnumSchema, err error) {

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	cies = &CommandInputEnumSchema{}

	switch original.(type) {

	case map[string]interface{}:

		cies.EnumSchema, err = NewEnumSchemaFromInterface(original, context)
		if err != nil {
			err = fmt.Errorf("(NewCommandInputEnumSchemaFromInterface) NewEnumSchemaFromInterface returns: %s", err.Error())
			return
		}

		original_map, _ := original.(map[string]interface{})

		inputBinding, has_inputBinding := original_map["inputBinding"]

		if has_inputBinding {

			var clb *CommandLineBinding
			clb, err = NewCommandLineBinding(inputBinding, context)
			if err != nil {
				err = fmt.Errorf("(NewCommandInputEnumSchemaFromInterface) NewCommandLineBinding returned: %s", err.Error())
				return
			}

			cies.InputBinding = clb
		}

		name_if, has_name := original_map["name"]
		if has_name {
			name_str, ok := name_if.(string)
			if !ok {
				err = fmt.Errorf("(NewCommandInputEnumSchemaFromInterface) field name is not string")
				return
			}
			cies.Name = name_str
		}

	default:
		err = fmt.Errorf("(NewCommandInputEnumSchemaFromInterface) error")
		return

	}
	return
}
