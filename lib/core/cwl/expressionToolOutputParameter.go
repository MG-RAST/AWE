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
func NewExpressionToolOutputParameter(original interface{}, thisID string, schemata []CWLType_Type, context *WorkflowContext) (wop *ExpressionToolOutputParameter, err error) {
	var outputParameter ExpressionToolOutputParameter

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	switch original.(type) {
	case string:
		err = fmt.Errorf("(NewExpressionToolOutputParameter) type string not supported!! ")
		return
	case map[string]interface{}:

		var op *OutputParameter
		var opIf interface{}
		opIf, err = NewOutputParameterFromInterface(original, thisID, schemata, "Output", context)
		if err != nil {
			err = fmt.Errorf("(NewExpressionToolOutputParameter) NewOutputParameterFromInterface returns %s", err.Error())
			return
		}

		op, ok := opIf.(*OutputParameter)
		if !ok {
			err = fmt.Errorf("(NewExpressionToolOutputParameter) could not cast into *OutputParameter")
			return
		}

		err = mapstructure.Decode(original, &outputParameter)
		if err != nil {
			err = fmt.Errorf("(NewExpressionToolOutputParameter) decode error: %s", err.Error())
			return
		}
		wop = &outputParameter

		wop.OutputParameter = *op
	default:
		err = fmt.Errorf("(NewExpressionToolOutputParameter) type unknown, %s", reflect.TypeOf(original))
		return

	}

	return
}

// NewExpressionToolOutputParameterArray _
func NewExpressionToolOutputParameterArray(original interface{}, schemata []CWLType_Type, context *WorkflowContext) (newArray []interface{}, err error) {

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	newArray = []interface{}{}
	switch original.(type) {
	case map[string]interface{}:
		for k, v := range original.(map[string]interface{}) {
			//fmt.Printf("A")
			var elementStr string
			var ok bool
			elementStr, ok = v.(string)

			if ok {
				var result CWLType_Type
				result, err = NewCWLType_TypeFromString(schemata, elementStr, "ExpressionToolOutput")
				if err != nil {
					err = fmt.Errorf("(NewExpressionToolOutputParameterArray) NewCWLType_TypeFromString returns: %s", err.Error())
					return
				}

				etop := ExpressionToolOutputParameter{}
				etop.Id = k
				etop.Type = result

				newArray = append(newArray, result)
				continue
			}
			var outputParameter *ExpressionToolOutputParameter
			outputParameter, err = NewExpressionToolOutputParameter(v, k, schemata, context)
			if err != nil {
				err = fmt.Errorf("(NewExpressionToolOutputParameterArray) A) NewExpressionToolOutputParameter returns: %s", err.Error())

				return
			}
			outputParameter.Id = k
			//fmt.Printf("C")
			newArray = append(newArray, *outputParameter)
			//fmt.Printf("D")

		}

		return
	case []interface{}:

		for _, v := range original.([]interface{}) {
			//fmt.Printf("A")

			var elementStr string
			var ok bool
			elementStr, ok = v.(string)

			if ok {
				var result CWLType_Type
				result, err = NewCWLType_TypeFromString(schemata, elementStr, "ExpressionToolOutput")
				if err != nil {
					err = fmt.Errorf("(NewExpressionToolOutputParameterArray) NewCWLType_TypeFromString returns: %s", err.Error())
					return
				}
				newArray = append(newArray, result)
				continue
			}

			var outputParameter *ExpressionToolOutputParameter
			outputParameter, err = NewExpressionToolOutputParameter(v, "", schemata, context)
			if err != nil {
				err = fmt.Errorf("(NewExpressionToolOutputParameterArray) B) NewExpressionToolOutputParameter returns: %s", err.Error())
				return
			}
			//output_parameter.Id = k.(string)
			//fmt.Printf("C")
			newArray = append(newArray, *outputParameter)
			//fmt.Printf("D")

		}

		return

	default:
		spew.Dump(newArray)
		err = fmt.Errorf("(NewExpressionToolOutputParameterArray) type %s unknown", reflect.TypeOf(original))
	}
	//spew.Dump(new_array)
	return
}
