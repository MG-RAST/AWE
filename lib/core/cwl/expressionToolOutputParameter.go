package cwl

import (
	"fmt"
	"reflect"

	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
)

// http://www.commonwl.org/v1.0/Workflow.html#ExpressionToolOutputParameter
type ExpressionToolOutputParameter struct {
	OutputParameter `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
}

// type: CWLType | OutputRecordSchema | OutputEnumSchema | OutputArraySchema | string | array<CWLType | OutputRecordSchema | OutputEnumSchema | OutputArraySchema | string>

func NewExpressionToolOutputParameter(original interface{}, schemata []CWLType_Type) (wop *ExpressionToolOutputParameter, err error) {
	var output_parameter ExpressionToolOutputParameter

	original, err = MakeStringMap(original)
	if err != nil {
		return
	}

	switch original.(type) {

	case map[string]interface{}:
		original_map, ok := original.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewExpressionToolOutputParameter) type switch error")
			return
		}

		err = NormalizeOutputParameter(original_map)
		if err != nil {
			err = fmt.Errorf("(NewExpressionToolOutputParameter) NormalizeOutputParameter returns %s", err.Error())
			return
		}

		wop_type, has_type := original_map["type"]
		if has_type {

			wop_type_array, xerr := NewWorkflowOutputParameterTypeArray(wop_type, schemata)
			if xerr != nil {
				err = fmt.Errorf("from NewWorkflowOutputParameterTypeArray: %s", xerr.Error())
				return
			}
			//fmt.Println("type of wop_type_array")
			//fmt.Println(reflect.TypeOf(wop_type_array))
			//fmt.Println("original:")
			//spew.Dump(original)
			//fmt.Println("wop_type_array:")
			//spew.Dump(wop_type_array)
			original_map["type"] = wop_type_array

		}

		err = mapstructure.Decode(original, &output_parameter)
		if err != nil {
			err = fmt.Errorf("(NewWorkflowOutputParameter) decode error: %s", err.Error())
			return
		}
		wop = &output_parameter
	default:
		err = fmt.Errorf("(NewWorkflowOutputParameter) type unknown, %s", reflect.TypeOf(original))
		return

	}

	return
}

func NewExpressionToolOutputParameterArray(original interface{}, schemata []CWLType_Type) (new_array_ptr *[]ExpressionToolOutputParameter, err error) {

	new_array := []ExpressionToolOutputParameter{}
	switch original.(type) {
	case map[interface{}]interface{}:
		for k, v := range original.(map[interface{}]interface{}) {
			//fmt.Printf("A")

			output_parameter, xerr := NewExpressionToolOutputParameter(v, schemata)
			if xerr != nil {
				err = xerr
				return
			}
			output_parameter.Id = k.(string)
			//fmt.Printf("C")
			new_array = append(new_array, *output_parameter)
			//fmt.Printf("D")

		}
		new_array_ptr = &new_array
		return
	case []interface{}:

		for _, v := range original.([]interface{}) {
			//fmt.Printf("A")

			output_parameter, xerr := NewExpressionToolOutputParameter(v, schemata)
			if xerr != nil {
				err = xerr
				return
			}
			//output_parameter.Id = k.(string)
			//fmt.Printf("C")
			new_array = append(new_array, *output_parameter)
			//fmt.Printf("D")

		}
		new_array_ptr = &new_array
		return

	default:
		spew.Dump(new_array)
		err = fmt.Errorf("(NewExpressionToolOutputParameterArray) type %s unknown", reflect.TypeOf(original))
	}
	//spew.Dump(new_array)
	return
}
