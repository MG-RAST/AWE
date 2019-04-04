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

	var op *OutputParameter
	op, err = NewOutputParameterFromInterface(original, schemata, "CommandOutput", context)
	if err != nil {
		err = fmt.Errorf("(NewCommandOutputParameter) NewOutputParameterFromInterface returns %s", err.Error())
		return
	}

	switch original.(type) {

	case map[string]interface{}:

		//original_map, ok := original.(map[string]interface{})
		//if !ok {
		//	err = fmt.Errorf("(NewCommandOutputParameter) type error")
		//	return
		//}

		// type:
		// any of CWLType | stdout | stderr | CommandOutputRecordSchema | CommandOutputEnumSchema | CommandOutputArraySchema | string | array<CWLType | CommandOutputRecordSchema | CommandOutputEnumSchema | CommandOutputArraySchema | string>
		// COPtype, ok := original_map["type"]
		// if ok {
		// 	original_map["type"], err = NewCommandOutputParameterTypeArray(COPtype, schemata)
		// 	if err != nil {
		// 		fmt.Println("NewCommandOutputParameter:")
		// 		spew.Dump(original_map)
		// 		err = fmt.Errorf("(NewCommandOutputParameter) NewCommandOutputParameterTypeArray returns %s", err.Error())
		// 		return
		// 	}
		// }

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
func NewCommandOutputParameterArray(original interface{}, schemata []CWLType_Type, context *WorkflowContext) (copa *[]CommandOutputParameter, err error) {

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
		copa = &[]CommandOutputParameter{*cop}
	case []interface{}:
		copaNptr := []CommandOutputParameter{}

		originalArray := original.([]interface{})

		for _, element := range originalArray {
			cop, xerr := NewCommandOutputParameter(element, schemata, context)
			if xerr != nil {
				err = fmt.Errorf("(NewCommandOutputParameterArray) b NewCommandOutputParameter returns: %s", xerr.Error())
				return
			}
			copaNptr = append(copaNptr, *cop)
		}

		copa = &copaNptr
	default:
		err = fmt.Errorf("NewCommandOutputParameterArray, unknown type %s", reflect.TypeOf(original))
	}
	return

}
