package cwl

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"

	"reflect"

	"github.com/mitchellh/mapstructure"
)

//http://www.commonwl.org/v1.0/Workflow.html#WorkflowStepInput
type WorkflowStepInput struct {
	CWLObjectImpl `yaml:",inline" bson:",inline" json:",inline" mapstructure:",squash"`
	Id            string           `yaml:"id,omitempty" bson:"id,omitempty" json:"id,omitempty" mapstructure:"id,omitempty"`
	Source        interface{}      `yaml:"source,omitempty" bson:"source,omitempty" json:"source,omitempty" mapstructure:"source,omitempty"` // MultipleInputFeatureRequirement, multiple inbound data links listed in the source field
	Source_index  int              `yaml:"source_index,omitempty" bson:"source_index,omitempty" json:"source_index,omitempty" mapstructure:"source_index,omitempty"`
	LinkMerge     *LinkMergeMethod `yaml:"linkMerge,omitempty" bson:"linkMerge,omitempty" json:"linkMerge,omitempty" mapstructure:"linkMerge,omitempty"`
	Default       interface{}      `yaml:"default,omitempty" bson:"default,omitempty" json:"default,omitempty" mapstructure:"default,omitempty"`         // type Any does not make sense
	ValueFrom     Expression       `yaml:"valueFrom,omitempty" bson:"valueFrom,omitempty" json:"valueFrom,omitempty" mapstructure:"valueFrom,omitempty"` // StepInputExpressionRequirement
	Ready         bool             `yaml:"-" bson:"-" json:"-" mapstructure:"-"`
}

func (w WorkflowStepInput) GetClass() string {
	return "WorkflowStepInput"
}
func (w WorkflowStepInput) GetID() string   { return w.Id }
func (w WorkflowStepInput) SetID(id string) { w.Id = id }

func (w WorkflowStepInput) IsCWLMinimal() {}

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

func NewWorkflowStepInput(original interface{}, context *WorkflowContext) (input_parameter_ptr *WorkflowStepInput, err error) {

	input_parameter := WorkflowStepInput{}
	input_parameter_ptr = &input_parameter

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	switch original.(type) {
	case string:

		source_string := original.(string)
		input_parameter.Source = []string{source_string}
		return

	case int:
		//fmt.Println(CWLInt)
		original_int := original.(int)
		input_parameter.Default, err = NewInt(original_int, context) // input_parameter.Id
		if err != nil {
			err = fmt.Errorf("(NewCWLType) NewInt: %s", err.Error())
			return
		}
		return

	case map[string]interface{}:
		//fmt.Println("case map[string]interface{}")

		original_map := original.(map[string]interface{})

		//source, ok := original_map["source"]

		//if ok {
		//	source_str, ok := source.(string)
		//	if ok {
		//		original_map["source"] = []string{source_str}
		//	}
		//}

		default_value, has_default := original_map["default"]
		if has_default {
			var any CWLType
			//fmt.Println("trying:")
			//spew.Dump(original)
			any, err = NewCWLType("", default_value, context)
			if err != nil {
				fmt.Println("problematic Default:")
				spew.Dump(original)
				err = fmt.Errorf("(NewWorkflowStepInput) NewCWLType returned: %s", err.Error())
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

		if input_parameter_ptr.LinkMerge == nil {
			// handle special case where LinkMerge is not defined, source is an array of length 1
			// https://github.com/common-workflow-language/common-workflow-language/issues/675

			switch input_parameter_ptr.Source.(type) {
			case []interface{}:
				var source_array []interface{}
				source_array, ok = input_parameter_ptr.Source.([]interface{})
				if !ok {
					err = fmt.Errorf("(NewWorkflowStepInput) could not convert to []interface{}")
					return
				}

				if len(source_array) == 1 {
					input_parameter_ptr.Source = source_array[0]
				}

			}

		}

	default:

		err = fmt.Errorf("(NewWorkflowStepInput) Input type %s can not be parsed", reflect.TypeOf(original))

	}

	return
}

// func (input WorkflowStepInput) GetObject(c *CWL_collection) (obj *CWLObject, err error) {
//
// 	var cwl_obj CWLObject
//
// 	if len(input.Source) > 0 {
// 		err = fmt.Errorf("Source is defined and should be used")
// 	} else if string(input.ValueFrom) != "" {
// 		new_string := string(input.ValueFrom)
// 		evaluated_string := c.Evaluate(new_string)
//
// 		cwl_obj = NewString(input.Id, evaluated_string) // TODO evaluate here !!!!! get helper
// 	} else if input.Default != nil {
// 		var any Any
// 		any, err = NewAny(input.Default)
// 		if err != nil {
// 			err = fmt.Errorf("(GetObject) NewAny returns", err.Error())
// 			return
// 		}
// 		cwl_obj = any
// 	} else {
// 		err = fmt.Errorf("no input (source, default or valueFrom) defined for %s", input.Id)
// 	}
// 	obj = &cwl_obj
// 	return
// }

func CreateWorkflowStepInputArray(original interface{}, context *WorkflowContext) (array_ptr *[]WorkflowStepInput, err error) {

	array := []WorkflowStepInput{}
	array_ptr = &array

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	switch original.(type) {
	case map[string]interface{}:
		original_map := original.(map[string]interface{})

		for k, v := range original_map {
			//v is an input

			input_parameter, xerr := NewWorkflowStepInput(v, context)
			if xerr != nil {
				err = fmt.Errorf("(CreateWorkflowStepInputArray) (map) NewWorkflowStepInput returns: %s", xerr.Error())
				return
			}

			if input_parameter.Id == "" {
				input_parameter.Id = k
			}

			array = append(array, *input_parameter)
			//fmt.Printf("D")
		}
		return

	case []interface{}:

		for _, v := range original.([]interface{}) {
			//v is an input

			input_parameter, xerr := NewWorkflowStepInput(v, context)
			if xerr != nil {
				err = fmt.Errorf("(CreateWorkflowStepInputArray) (array) NewWorkflowStepInput returns: %s", xerr.Error())
				return
			}

			array = append(array, *input_parameter)
			//fmt.Printf("D")
		}
		return

	default:
		err = fmt.Errorf("(CreateWorkflowStepInputArray) type unknown: %s", reflect.TypeOf(original))

	}
	//spew.Dump(new_array)
	return
}
