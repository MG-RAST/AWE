package cwl

import (
	"fmt"
	//"github.com/davecgh/go-spew/spew"

	"github.com/mitchellh/mapstructure"
	"reflect"
)

//http://www.commonwl.org/v1.0/Workflow.html#WorkflowStepInput
type WorkflowStepInput struct {
	Id        string           `yaml:"id,omitempty" bson:"id,omitempty" json:"id,omitempty"`
	Source    []string         `yaml:"source,omitempty" bson:"source,omitempty" json:"source,omitempty"` // MultipleInputFeatureRequirement
	LinkMerge *LinkMergeMethod `yaml:"linkMerge,omitempty" bson:"linkMerge,omitempty" json:"linkMerge,omitempty"`
	Default   interface{}      `yaml:"default,omitempty" bson:"default,omitempty" json:"default,omitempty"`       // type Any does not make sense
	ValueFrom Expression       `yaml:"valueFrom,omitempty" bson:"valueFrom,omitempty" json:"valueFrom,omitempty"` // StepInputExpressionRequirement
	Ready     bool             `yaml:"-" bson:"-" json:"-"`
}

func (w WorkflowStepInput) GetClass() CWLType_Type {
	return CWLType_Type("WorkflowStepInput")
}
func (w WorkflowStepInput) GetId() string   { return w.Id }
func (w WorkflowStepInput) SetId(id string) { w.Id = id }

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

	original, err = makeStringMap(original)
	if err != nil {
		return
	}

	switch original.(type) {
	case string:

		source_string := original.(string)
		input_parameter.Source = []string{"#" + source_string}
		return

	case int:
		fmt.Println(CWL_int)
		original_int := original.(int)
		input_parameter.Default = NewInt(input_parameter.Id, original_int)
		return

	case map[string]interface{}:
		fmt.Println("case map[string]interface{}")

		original_map := original.(map[string]interface{})

		source, ok := original_map["source"]

		if ok {
			source_str, ok := source.(string)
			if ok {
				original_map["source"] = []string{source_str}
			}
		}

		default_value, has_default := original_map["default"]
		if has_default {
			var any Any
			any, err = NewAny(default_value)
			if err != nil {
				err = fmt.Errorf("(NewWorkflowStepInput) NewAny returned: %s", err.Error())
				return
			}
			original_map["default"] = any
		}

		err = mapstructure.Decode(original, input_parameter_ptr)
		if err != nil {
			err = fmt.Errorf("(NewWorkflowStepInput) %s", err.Error())
			return
		}

		// TODO would it be better to do it later?

		// set Default field
		// default_value, ok := original_map["default"]
		// 	//default_value, ok := , errx := GetMapElement(original_map, "default")
		// 	if ok {
		// 		switch default_value.(type) {
		// 		case string:
		// 			input_parameter.Default = NewString(input_parameter.Id, default_value.(string))
		// 		case int:
		// 			input_parameter.Default = NewInt(input_parameter.Id, default_value.(int))
		// 		default:
		// 			err = fmt.Errorf("(NewWorkflowStepInput) string or int expected for key \"default\", got %s ", reflect.TypeOf(default_value))
		// 			return
		// 		}
		// 	}

		// set ValueFrom field
		//valueFrom_if, errx := GetMapElement(original.(map[interface{}]interface{}), "valueFrom")
		valueFrom_if, ok := original_map["valueFrom"]
		if ok {
			valueFrom_str, ok := valueFrom_if.(string)
			if !ok {
				err = fmt.Errorf("(NewWorkflowStepInput) cannot convert valueFrom")
				return
			}
			input_parameter.ValueFrom = Expression(valueFrom_str)
		}
		return

	default:

		err = fmt.Errorf("(NewWorkflowStepInput) Input type %s can not be parsed", reflect.TypeOf(original))
		return
	}

	return
}

func (input WorkflowStepInput) GetObject(c *CWL_collection) (obj *CWL_object, err error) {

	var cwl_obj CWL_object

	if len(input.Source) > 0 {
		err = fmt.Errorf("Source is defined and should be used")
	} else if string(input.ValueFrom) != "" {
		new_string := string(input.ValueFrom)
		evaluated_string := c.Evaluate(new_string)

		cwl_obj = NewString(input.Id, evaluated_string) // TODO evaluate here !!!!! get helper
	} else if input.Default != nil {
		var any Any
		any, err = NewAny(input.Default)
		if err != nil {
			return
		}
		cwl_obj = any
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
