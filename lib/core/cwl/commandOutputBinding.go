package cwl

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
)

type CommandOutputBinding struct {
	Glob         []Expression `yaml:"glob"`
	LoadContents bool         `yaml:"loadContents"`
	OutputEval   Expression   `yaml:"outputEval"`
}

func NewCommandOutputBinding(original interface{}) (commandOutputBinding *CommandOutputBinding, err error) {

	switch original.(type) {
	case map[interface{}]interface{}:
		original_map := original.(map[interface{}]interface{})

		glob, ok := original_map["glob"]
		if ok {
			original_map["glob"], err = NewExpressionArray(glob)
			if err != nil {
				return
			}
		}
		outputEval, ok := original_map["outputEval"]
		if ok {
			original_map["outputEval"], err = NewExpression(outputEval)
			if err != nil {
				return
			}
		}
	default:
		err = fmt.Errorf("NewCommandOutputBinding: tyype unknown")
		return
	}

	commandOutputBinding = &CommandOutputBinding{}
	err = mapstructure.Decode(original, &commandOutputBinding)
	if err != nil {
		err = fmt.Errorf("(NewCommandOutputBinding) %s", err.Error())
		return
	}
	//output_parameter.OutputBinding = outputBinding

	return
}
