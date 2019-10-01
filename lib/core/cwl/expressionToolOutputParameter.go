package cwl

import (
	"fmt"
	"reflect"
)

// ExpressionToolOutputParameter http://www.commonwl.org/v1.0/Workflow.html#ExpressionToolOutputParameter
type ExpressionToolOutputParameter struct {
	OutputParameter `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // provides Id, Label, SecondaryFiles, Format, Streamable, OutputBinding, Type
}

// type: CWLType | OutputRecordSchema | OutputEnumSchema | OutputArraySchema | string | array<CWLType | OutputRecordSchema | OutputEnumSchema | OutputArraySchema | string>

//NewExpressionToolOutputParameter _
func NewExpressionToolOutputParameter(original interface{}, thisID string, schemata []CWLType_Type, context *WorkflowContext) (etop *ExpressionToolOutputParameter, err error) {
	//var outputParameter ExpressionToolOutputParameter
	etop = &ExpressionToolOutputParameter{}

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	var op *OutputParameter
	op, err = NewOutputParameterFromInterface(original, thisID, schemata, "ExpressionToolOutput", context)
	if err != nil {
		err = fmt.Errorf("(NewExpressionToolOutputParameter) NewOutputParameterFromInterface returns %s", err.Error())
		return
	}

	switch original.(type) {
	case string:

		etop.OutputParameter = *op

	case map[string]interface{}:

		// not needed as there are no additonal fields

		// err = mapstructure.Decode(original, &outputParameter)
		// if err != nil {
		// 	err = fmt.Errorf("(NewExpressionToolOutputParameter) decode error: %s", err.Error())
		// 	return
		// }
		// wop = &outputParameter

		etop.OutputParameter = *op
	default:
		err = fmt.Errorf("(NewExpressionToolOutputParameter) type unknown, %s", reflect.TypeOf(original))
		return

	}

	if etop == nil {
		err = fmt.Errorf("(NewExpressionToolOutputParameter) etop ==nil")
		return
	}

	return
}

// NewExpressionToolOutputParameterMap _
func NewExpressionToolOutputParameterMap(original interface{}, schemata []CWLType_Type, context *WorkflowContext) (newMap map[string]interface{}, err error) {

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	newMap = make(map[string]interface{})

	//newArray = []interface{}{}
	switch original.(type) {
	case map[string]interface{}:
		originalMap := original.(map[string]interface{})
		for k, v := range originalMap {
			//fmt.Printf("A")

			var outputParameter *ExpressionToolOutputParameter
			outputParameter, err = NewExpressionToolOutputParameter(v, k, schemata, context)
			if err != nil {
				err = fmt.Errorf("(NewExpressionToolOutputParameterMap) A) NewExpressionToolOutputParameter returns: %s", err.Error())

				return
			}

			outputParameter.Id = k
			//fmt.Printf("C")
			newMap[k] = outputParameter
			//fmt.Printf("D")

		}

		return
	case []interface{}:

		for _, v := range original.([]interface{}) {
			//fmt.Printf("A")

			var outputParameter *ExpressionToolOutputParameter
			outputParameter, err = NewExpressionToolOutputParameter(v, "", schemata, context)
			if err != nil {
				err = fmt.Errorf("(NewExpressionToolOutputParameterMap) B) NewExpressionToolOutputParameter returns: %s", err.Error())
				return
			}
			thisID := outputParameter.Id
			if thisID == "" {
				err = fmt.Errorf("(NewExpressionToolOutputParameterMap) outputParameter has no id !")
				return
			}
			//output_parameter.Id = k.(string)
			//fmt.Printf("C")
			//newArray = append(newArray, *outputParameter)
			newMap[thisID] = outputParameter
			//fmt.Printf("D")

		}

		return

	default:

		err = fmt.Errorf("(NewExpressionToolOutputParameterMap) type %s unknown", reflect.TypeOf(original))
	}
	//spew.Dump(new_array)
	return
}
