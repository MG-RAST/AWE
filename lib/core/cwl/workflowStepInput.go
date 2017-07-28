package cwl

import (
	"fmt"
	//"github.com/davecgh/go-spew/spew"
	cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	"github.com/mitchellh/mapstructure"
)

//http://www.commonwl.org/v1.0/Workflow.html#WorkflowStepInput
type WorkflowStepInput struct {
	Id        string               `yaml:"id"`
	Source    []string             `yaml:"source"` // MultipleInputFeatureRequirement
	LinkMerge LinkMergeMethod      `yaml:"linkMerge"`
	Default   cwl_types.Any        `yaml:"default"`   // type Any does not make sense
	ValueFrom cwl_types.Expression `yaml:"valueFrom"` // StepInputExpressionRequirement
	Ready     bool                 `yaml:"-"`
}

func (w WorkflowStepInput) GetClass() string { return "WorkflowStepInput" }
func (w WorkflowStepInput) GetId() string    { return w.Id }
func (w WorkflowStepInput) SetId(id string)  { w.Id = id }

func (w WorkflowStepInput) Is_CWL_minimal() {}

//func (input WorkflowStepInput) GetString() (value string, err error) {
//	if len(input.Source) > 0 {
//		err = fmt.Errorf("Source is defined and should be used")
//	} else if string(input.ValueFrom) != "" {
//		value = string(input.ValueFrom)
//	} else if input.Default != nil {
//		value = input.Default
//	} else {
//		err = fmt.Errorf("no input (source, default or valueFrom) defined for %s", id)
//	}
//	return
//}

func NewWorkflowStepInput(original interface{}) (input_parameter_ptr *WorkflowStepInput, err error) {

	input_parameter := WorkflowStepInput{}
	input_parameter_ptr = &input_parameter

	switch original.(type) {
	case string:

		source_string := original.(string)
		input_parameter.Source = []string{"#" + source_string}
		return

	case int:
		fmt.Println(cwl_types.CWL_int)
		input_parameter.Default = &cwl_types.Int{Id: input_parameter.Id, Value: original.(int)}
		return

	case map[interface{}]interface{}:
		fmt.Println("case map[interface{}]interface{}")

		original_map := original.(map[interface{}]interface{})

		source, ok := original_map["source"]

		if ok {
			source_str, ok := source.(string)
			if ok {
				original_map["source"] = []string{source_str}
			}
		}

		err = mapstructure.Decode(original, input_parameter_ptr)
		if err != nil {
			err = fmt.Errorf("(NewWorkflowStepInput) %s", err.Error())
			return
		}

		// TODO would it be better to do it later?

		// set Default field
		default_value, errx := GetMapElement(original.(map[interface{}]interface{}), "default")
		if errx == nil {
			switch default_value.(type) {
			case string:
				input_parameter.Default = &cwl_types.String{Id: input_parameter.Id, Value: default_value.(string)}
			case int:
				input_parameter.Default = &cwl_types.Int{Id: input_parameter.Id, Value: default_value.(int)}
			default:
				err = fmt.Errorf("(NewWorkflowStepInput) string or int expected for key \"default\"")
				return
			}
		}

		// set ValueFrom field
		valueFrom_if, errx := GetMapElement(original.(map[interface{}]interface{}), "valueFrom")
		if errx == nil {
			valueFrom_str, ok := valueFrom_if.(string)
			if !ok {
				err = fmt.Errorf("(NewWorkflowStepInput) cannot convert valueFrom")
				return
			}
			input_parameter.ValueFrom = cwl_types.Expression(valueFrom_str)
		}
		return

	default:

		err = fmt.Errorf("(NewWorkflowStepInput) Input type for %s can not be parsed", input_parameter.Id)
		return
	}

	return
}

func (input WorkflowStepInput) GetObject(c *CWL_collection) (obj *cwl_types.CWL_object, err error) {

	var cwl_obj cwl_types.CWL_object

	if len(input.Source) > 0 {
		err = fmt.Errorf("Source is defined and should be used")
	} else if string(input.ValueFrom) != "" {
		new_string := string(input.ValueFrom)
		evaluated_string := c.Evaluate(new_string)

		cwl_obj = cwl_types.NewString(input.Id, evaluated_string) // TODO evaluate here !!!!! get helper
	} else if input.Default != nil {
		cwl_obj = input.Default
	} else {
		err = fmt.Errorf("no input (source, default or valueFrom) defined for %s", input.Id)
	}
	obj = &cwl_obj
	return
}

func CreateWorkflowStepInputArray(original interface{}) (array_ptr *[]WorkflowStepInput, err error) {

	array := []WorkflowStepInput{}
	array_ptr = &array

	switch original.(type) {
	case map[interface{}]interface{}:
		original_map := original.(map[interface{}]interface{})

		for k, v := range original_map {
			//v is an input

			input_parameter, xerr := NewWorkflowStepInput(v)
			if xerr != nil {
				err = xerr
				return
			}

			if input_parameter.Id == "" {
				input_parameter.Id = k.(string)
			}

			array = append(array, *input_parameter)
			//fmt.Printf("D")
		}
		return

	case []interface{}:

		for _, v := range original.([]interface{}) {
			//v is an input

			input_parameter, xerr := NewWorkflowStepInput(v)
			if xerr != nil {
				err = xerr
				return
			}

			array = append(array, *input_parameter)
			//fmt.Printf("D")
		}
		return

	default:
		err = fmt.Errorf("(CreateWorkflowStepInputArray) type unknown")
		return

	}
	//spew.Dump(new_array)
	return
}
