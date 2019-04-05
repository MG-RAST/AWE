package cwl

import (
	"fmt"
	"reflect"

	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
)

// ExpressionToolOutputParameter http://www.commonwl.org/v1.0/Workflow.html#ExpressionToolOutputParameter
type ExpressionToolOutputParameter struct {
	OutputParameter `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // provides Id, Label, SecondaryFiles, Format, Streamable, OutputBinding, Type
}

// type: CWLType | OutputRecordSchema | OutputEnumSchema | OutputArraySchema | string | array<CWLType | OutputRecordSchema | OutputEnumSchema | OutputArraySchema | string>

//NewExpressionToolOutputParameter _
func NewExpressionToolOutputParameter(original interface{}, schemata []CWLType_Type, context *WorkflowContext) (wop *ExpressionToolOutputParameter, err error) {
	var outputParameter ExpressionToolOutputParameter

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	var op *OutputParameter
	op, err = NewOutputParameterFromInterface(original, schemata, "Output", context)
	if err != nil {
		err = fmt.Errorf("(NewExpressionToolOutputParameter) NewOutputParameterFromInterface returns %s", err.Error())
		return
	}

	switch original.(type) {

	case map[string]interface{}:

		err = mapstructure.Decode(original, &outputParameter)
		if err != nil {
			err = fmt.Errorf("(NewWorkflowOutputParameter) decode error: %s", err.Error())
			return
		}
		wop = &outputParameter

		wop.OutputParameter = *op
	default:
		err = fmt.Errorf("(NewWorkflowOutputParameter) type unknown, %s", reflect.TypeOf(original))
		return

	}

	return
}

// NewExpressionToolOutputParameterArray _
func NewExpressionToolOutputParameterArray(original interface{}, schemata []CWLType_Type, context *WorkflowContext) (new_array_ptr *[]ExpressionToolOutputParameter, err error) {

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	newArray := []ExpressionToolOutputParameter{}
	switch original.(type) {
	case map[string]interface{}:
		for k, v := range original.(map[string]interface{}) {
			//fmt.Printf("A")

			outputParameter, xerr := NewExpressionToolOutputParameter(v, schemata, context)
			if xerr != nil {
				err = xerr
				return
			}
			outputParameter.Id = k
			//fmt.Printf("C")
			newArray = append(newArray, *outputParameter)
			//fmt.Printf("D")

		}
		new_array_ptr = &newArray
		return
	case []interface{}:

		for _, v := range original.([]interface{}) {
			//fmt.Printf("A")

			outputParameter, xerr := NewExpressionToolOutputParameter(v, schemata, context)
			if xerr != nil {
				err = xerr
				return
			}
			//output_parameter.Id = k.(string)
			//fmt.Printf("C")
			newArray = append(newArray, *outputParameter)
			//fmt.Printf("D")

		}
		new_array_ptr = &newArray
		return

	default:
		spew.Dump(newArray)
		err = fmt.Errorf("(NewExpressionToolOutputParameterArray) type %s unknown", reflect.TypeOf(original))
	}
	//spew.Dump(new_array)
	return
}
