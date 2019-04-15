package cwl

import (
	"fmt"
	"reflect"

	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
)

// CommandOutputParameter https://www.commonwl.org/v1.0/CommandLineTool.html#CommandOutputParameter
type CommandOutputParameter struct {
	OutputParameter `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // provides Id, Label, SecondaryFiles, Format, Streamable, OutputBinding, Type

	Description string `yaml:"description,omitempty" bson:"description,omitempty" json:"description,omitempty" mapstructure:"description,omitempty"`
}

// NewCommandOutputParameter _
func NewCommandOutputParameter(original interface{}, schemata []CWLType_Type, context *WorkflowContext) (output_parameter *CommandOutputParameter, err error) {

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	switch original.(type) {

	case string:

	case map[string]interface{}:

		var op *OutputParameter
		var opIf interface{}
		opIf, err = NewOutputParameterFromInterface(original, schemata, "CommandOutput", context)
		if err != nil {
			err = fmt.Errorf("(NewCommandOutputParameter) NewOutputParameterFromInterface returns %s", err.Error())
			return
		}

		op, ok := opIf.(*OutputParameter)
		if !ok {
			err = fmt.Errorf("(NewCommandOutputParameter) could not cast into *OutputParameter")
			return
		}

		output_parameter = &CommandOutputParameter{}
		err = mapstructure.Decode(original, output_parameter)
		if err != nil {
			err = fmt.Errorf("(NewCommandOutputParameter) mapstructure returned: %s", err.Error())
			return
		}

		output_parameter.OutputParameter = *op
	default:
		spew.Dump(original)
		err = fmt.Errorf("NewCommandOutputParameter, unknown type %s", reflect.TypeOf(original))
	}
	//spew.Dump(new_array)
	return
}

// NewCommandOutputParameterArray _
func NewCommandOutputParameterArray(original interface{}, schemata []CWLType_Type, context *WorkflowContext) (copa []interface{}, err error) {

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	switch original.(type) {
	case map[string]interface{}:
		cop, xerr := NewCommandOutputParameter(original, schemata, context)
		if xerr != nil {
			err = fmt.Errorf("(NewCommandOutputParameterArray) a NewCommandOutputParameter returns: %s", xerr.Error())
			return
		}
		copa = []interface{}{*cop}
	case []interface{}:
		copa = []interface{}{}

		originalArray := original.([]interface{})

		for _, element := range originalArray {
			var elementStr string
			var ok bool
			elementStr, ok = element.(string)

			if ok {
				var result CWLType_Type
				result, err = NewCWLType_TypeFromString(schemata, elementStr, "CommandOutput")
				if err != nil {
					err = fmt.Errorf("(NewCommandOutputParameterArray) NewCWLType_TypeFromString returns: %s", err.Error())
					return
				}
				copa = append(copa, result)
				continue
			}
			var cop *CommandOutputParameter
			cop, err = NewCommandOutputParameter(element, schemata, context)
			if err != nil {
				err = fmt.Errorf("(NewCommandOutputParameterArray) b NewCommandOutputParameter returns: %s", err.Error())
				return
			}
			copa = append(copa, *cop)
		}

	default:
		err = fmt.Errorf("NewCommandOutputParameterArray, unknown type %s", reflect.TypeOf(original))
	}
	return

}
