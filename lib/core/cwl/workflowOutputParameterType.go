package cwl

import (
	"fmt"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
	"reflect"
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
		output_type, ok := original_map["type"]

		if !ok {
			fmt.Printf("unknown type")
			spew.Dump(original)
			err = fmt.Errorf("(NewWorkflowOutputParameterType) Map-Type unknown")
		}

		switch output_type {
		case "record":
			result = OutputRecordSchema{}
			return
		case "enum":
			result = OutputEnumSchema{}
			return
		case "array":
			result = OutputArraySchema{}
			return
		default:
			err = fmt.Errorf("(NewWorkflowOutputParameterType) type %s is unknown", output_type)

		}

	default:
		err = fmt.Errorf("(NewWorkflowOutputParameterType) unknown type: %s", reflect.TypeOf(original))
		return
	}

	return
}

func NewWorkflowOutputParameterTypeArray(original interface{}, schemata []CWLType_Type) (result interface{}, err error) {

	wopta := []interface{}{}

	switch original.(type) {
	case map[interface{}]interface{}:

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
