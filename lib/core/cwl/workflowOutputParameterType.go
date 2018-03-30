package cwl

import (
	"fmt"
	"reflect"

	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
)

//type WorkflowOutputParameterType struct {
//	Type               string
//	OutputRecordSchema *OutputRecordSchema
//	OutputEnumSchema   *OutputEnumSchema
//	OutputArraySchema  *OutputArraySchema
//}

type OutputRecordSchema struct{}

type OutputEnumSchema struct{}

//type OutputArraySchema struct{}

// field type in http://www.commonwl.org/v1.0/Workflow.html#WorkflowOutputParameter
// CWLType | OutputRecordSchema | OutputEnumSchema | OutputArraySchema | string | array<CWLType | OutputRecordSchema | OutputEnumSchema | OutputArraySchema | string>

func NewWorkflowOutputParameterType(original interface{}, schemata []CWLType_Type) (result interface{}, err error) {
	//wopt := WorkflowOutputParameterType{}
	//wopt_ptr = &wopt

	original, err = MakeStringMap(original)
	if err != nil {
		return
	}

	switch original.(type) {
	case string:

		result_str := original.(string)

		result, err = NewCWLType_TypeFromString(schemata, result_str, "WorkflowOutput")
		if err != nil {
			return
		}

		return
	case map[string]interface{}:

		original_map := original.(map[string]interface{})
		output_type_if, ok := original_map["type"]

		if !ok {
			fmt.Printf("unknown type")
			spew.Dump(original)
			err = fmt.Errorf("(NewWorkflowOutputParameterType) Map-Type unknown")
			return
		}
		var output_type string
		output_type, ok = output_type_if.(string)
		if !ok {
			err = fmt.Errorf("(NewWorkflowOutputParameterType) value of field type is not string")
			return
		}

		if output_type == "" {
			err = fmt.Errorf("(NewWorkflowOutputParameterType) field \"type\" is empty")
			return
		}

		//fmt.Println("output_type: " + output_type)

		switch output_type {
		case "record":
			result = OutputRecordSchema{}
			err = fmt.Errorf("record not correctly implemented")
			return
		case "enum":
			result = OutputEnumSchema{}
			err = fmt.Errorf("enum not correctly implemented")
			return
		case "array":
			result = NewOutputArraySchema()
			return
		default:
			spew.Dump(original)
			err = fmt.Errorf("(NewWorkflowOutputParameterType) type %s is unknown", output_type)
			return
		}

	default:
		err = fmt.Errorf("(NewWorkflowOutputParameterType) unknown type: %s", reflect.TypeOf(original))

	}

	return
}

func NewWorkflowOutputParameterTypeArray(original interface{}, schemata []CWLType_Type) (result interface{}, err error) {

	original, err = MakeStringMap(original)
	if err != nil {
		return
	}

	wopta := []interface{}{}

	switch original.(type) {
	case map[string]interface{}:

		wopt, xerr := NewWorkflowOutputParameterType(original, schemata)
		if xerr != nil {
			err = xerr
			return
		}

		wopta = append(wopta, wopt)
		result = wopta
		return
	case []interface{}:
		logger.Debug(3, "[found array]")

		original_array := original.([]interface{})

		for _, element := range original_array {

			//spew.Dump(original)
			wopt, xerr := NewWorkflowOutputParameterType(element, schemata)
			if xerr != nil {
				err = xerr
				return
			}

			wopta = append(wopta, wopt)
		}

		result = wopta
		return
	case string:

		wopt, xerr := NewWorkflowOutputParameterType(original, schemata)
		if xerr != nil {
			err = xerr
			return
		}

		wopta = append(wopta, wopt)

		result = wopta
		return
	default:
		fmt.Printf("unknown type")
		spew.Dump(original)
		err = fmt.Errorf("(NewWorkflowOutputParameterTypeArray) unknown type")
	}
	return

}
