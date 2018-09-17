package cwl

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

// http://www.commonwl.org/v1.0/CommandLineTool.html#CommandOutputBinding
type CommandOutputBinding struct {
	Glob         []Expression `yaml:"glob,omitempty" bson:"glob,omitempty" json:"glob,omitempty"`
	LoadContents bool         `yaml:"loadContents,omitempty" bson:"loadContents,omitempty" json:"loadContents,omitempty"  mapstructure:"loadContents,omitempty"`
	OutputEval   *Expression  `yaml:"outputEval,omitempty" bson:"outputEval,omitempty" json:"outputEval,omitempty" mapstructure:"outputEval,omitempty"`
}

func NewCommandOutputBinding(original interface{}, context *WorkflowContext) (commandOutputBinding *CommandOutputBinding, err error) {

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	var outputEval *Expression

	switch original.(type) {

	case map[string]interface{}:
		var ok bool
		var original_map map[string]interface{}
		original_map, ok = original.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewCommandOutputBinding) type error b")
			return
		}

		var glob interface{}
		glob, ok = original_map["glob"]
		if ok {

			var glob_object *[]Expression
			glob_object, err = NewExpressionArray(glob)
			if err != nil {
				err = fmt.Errorf("(NewCommandOutputBinding/glob) NewExpressionArray returned: %s", err.Error())
				return
			}
			original_map["glob"] = glob_object
		}
		var outputEval_if interface{}
		outputEval_if, ok = original_map["outputEval"]
		if ok {
			outputEval, err = NewExpression(outputEval_if)
			if err != nil {
				return
			}
			delete(original_map, "outputEval")
			//original_map["outputEval"] = nil
		}
	default:
		err = fmt.Errorf("(NewCommandOutputBinding) type unknown")
		return
	}

	commandOutputBinding = &CommandOutputBinding{}
	err = mapstructure.Decode(original, &commandOutputBinding)
	if err != nil {
		err = fmt.Errorf("(NewCommandOutputBinding) mapstructure:  %s", err.Error())
		return
	}

	if outputEval != nil {
		commandOutputBinding.OutputEval = outputEval
	}

	//output_parameter.OutputBinding = outputBinding

	return
}
