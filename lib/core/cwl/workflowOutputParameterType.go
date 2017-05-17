package cwl

import (
	"fmt"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
)

type WorkflowOutputParameterType struct {
	Type               string
	OutputRecordSchema *OutputRecordSchema
	OutputEnumSchema   *OutputEnumSchema
	OutputArraySchema  *OutputArraySchema
}

type OutputRecordSchema struct{}

type OutputEnumSchema struct{}

type OutputArraySchema struct{}

func NewWorkflowOutputParameterType(original interface{}) (wopt_ptr *WorkflowOutputParameterType, err error) {
	wopt := WorkflowOutputParameterType{}
	wopt_ptr = &wopt

	switch original.(type) {
	case string:

		wopt.Type = original.(string)

		return
	case map[interface{}]interface{}:

		original_map := original.(map[interface{}]interface{})
		output_type, ok := original_map["type"]

		if !ok {
			fmt.Printf("unknown type")
			spew.Dump(original)
			err = fmt.Errorf("(NewWorkflowOutputParameterType) Map-Type unknown")
		}

		switch output_type {
		case "record":
			wopt.OutputRecordSchema = &OutputRecordSchema{}
		case "enum":
			wopt.OutputEnumSchema = &OutputEnumSchema{}
		case "array":
			wopt.OutputArraySchema = &OutputArraySchema{}
		default:
			err = fmt.Errorf("(NewWorkflowOutputParameterType) type %s is unknown", output_type)

		}

	default:
		err = fmt.Errorf("(NewWorkflowOutputParameterType) unknown type")
		return
	}

	return
}

func NewWorkflowOutputParameterTypeArray(original interface{}) (wopta_ptr *[]WorkflowOutputParameterType, err error) {
	wopta := []WorkflowOutputParameterType{}
	switch original.(type) {
	case map[interface{}]interface{}:

		wopt, xerr := NewWorkflowOutputParameterType(original)
		if xerr != nil {
			err = xerr
			return
		}
		wopta = append(wopta, *wopt)
		wopta_ptr = &wopta
		return
	case []interface{}:
		logger.Debug(3, "[found array]")

		original_array := original.([]interface{})

		for _, element := range original_array {

			spew.Dump(original)
			wopt, xerr := NewWorkflowOutputParameterType(element)
			if xerr != nil {
				err = xerr
				return
			}
			wopta = append(wopta, *wopt)
		}

		wopta_ptr = &wopta
		return
	case string:

		wopt, xerr := NewWorkflowOutputParameterType(original)
		if xerr != nil {
			err = xerr
			return
		}
		wopta = append(wopta, *wopt)

		wopta_ptr = &wopta
		return
	default:
		fmt.Printf("unknown type")
		spew.Dump(original)
		err = fmt.Errorf("(NewWorkflowOutputParameterTypeArray) unknown type")
	}
	return

}
