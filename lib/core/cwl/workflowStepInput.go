package cwl

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"

	"reflect"

	"github.com/mitchellh/mapstructure"
)

//WorkflowStepInput http://www.commonwl.org/v1.0/Workflow.html#WorkflowStepInput
type WorkflowStepInput struct {
	CWLObjectImpl `yaml:",inline" bson:",inline" json:",inline" mapstructure:",squash"`
	ID            string           `yaml:"id,omitempty" bson:"id,omitempty" json:"id,omitempty" mapstructure:"id,omitempty"`
	Source        interface{}      `yaml:"source,omitempty" bson:"source,omitempty" json:"source,omitempty" mapstructure:"source,omitempty"` // string | array<string> ; MultipleInputFeatureRequirement, multiple inbound data links listed in the source field
	SourceIndex   int              `yaml:"source_index,omitempty" bson:"source_index,omitempty" json:"source_index,omitempty" mapstructure:"source_index,omitempty"`
	LinkMerge     *LinkMergeMethod `yaml:"linkMerge,omitempty" bson:"linkMerge,omitempty" json:"linkMerge,omitempty" mapstructure:"linkMerge,omitempty"`
	Default       interface{}      `yaml:"default,omitempty" bson:"default,omitempty" json:"default,omitempty" mapstructure:"default,omitempty"`         // type Any does not make sense
	ValueFrom     Expression       `yaml:"valueFrom,omitempty" bson:"valueFrom,omitempty" json:"valueFrom,omitempty" mapstructure:"valueFrom,omitempty"` // StepInputExpressionRequirement
	Ready         bool             `yaml:"-" bson:"-" json:"-" mapstructure:"-"`
}

// GetClass _
func (w WorkflowStepInput) GetClass() string {
	return "WorkflowStepInput"
}

// GetID _
func (w WorkflowStepInput) GetID() string { return w.ID }

// SetID _
func (w WorkflowStepInput) SetID(id string) { w.ID = id }

// IsCWLMinimal _
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

// NewWorkflowStepInput _
func NewWorkflowStepInput(original interface{}, context *WorkflowContext) (inputParameterPtr *WorkflowStepInput, err error) {

	inputParameter := WorkflowStepInput{}
	inputParameterPtr = &inputParameter

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	switch original.(type) {
	case string:

		sourceString := original.(string)
		inputParameter.Source = []string{sourceString}
		return

	case []interface{}:

		originalStrArray := []string{}
		originalArray := original.([]interface{})
		for i := range originalArray {
			var value string
			var ok bool
			value, ok = originalArray[i].(string)
			if !ok {
				err = fmt.Errorf("(NewWorkflowStepInput) array elemenet not a string")
				return
			}
			originalStrArray = append(originalStrArray, value)

		}

		inputParameter.Source = originalStrArray

	case int:
		//fmt.Println(CWLInt)
		originalInt := original.(int)
		inputParameter.Default, err = NewInt(originalInt, context) // inputParameter.Id
		if err != nil {
			err = fmt.Errorf("(NewWorkflowStepInput) NewInt: %s", err.Error())
			return
		}
		return

	case map[string]interface{}:
		//fmt.Println("case map[string]interface{}")

		originalMap := original.(map[string]interface{})

		//source, ok := original_map["source"]

		//if ok {
		//	source_str, ok := source.(string)
		//	if ok {
		//		original_map["source"] = []string{source_str}
		//	}
		//}

		defaultValue, hasDefault := originalMap["default"]
		if hasDefault {
			var any CWLType
			//fmt.Println("trying:")
			//spew.Dump(original)
			any, err = NewCWLType("", "", defaultValue, context)
			if err != nil {
				fmt.Println("problematic Default:")
				spew.Dump(original)
				err = fmt.Errorf("(NewWorkflowStepInput) NewCWLType returned: %s", err.Error())
				return
			}
			originalMap["default"] = any
		}

		err = mapstructure.Decode(original, inputParameterPtr)
		if err != nil {
			err = fmt.Errorf("(NewWorkflowStepInput) %s", err.Error())
			return
		}

		// TODO would it be better to do it later?

		// set Default field
		// default_value, ok := originalMap["default"]
		// 	//default_value, ok := , errx := GetMapElement(originalMap, "default")
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
		valueFromIf, ok := originalMap["valueFrom"]
		if ok {
			valueFromStr, ok := valueFromIf.(string)
			if !ok {
				err = fmt.Errorf("(NewWorkflowStepInput) cannot convert valueFrom")
				return
			}
			inputParameter.ValueFrom = Expression(valueFromStr)
		}

		if inputParameterPtr.LinkMerge == nil {
			// handle special case where LinkMerge is not defined, source is an array of length 1
			// https://github.com/common-workflow-language/common-workflow-language/issues/675

			switch inputParameterPtr.Source.(type) {
			case []interface{}:
				var sourceArray []interface{}
				sourceArray, ok = inputParameterPtr.Source.([]interface{})
				if !ok {
					err = fmt.Errorf("(NewWorkflowStepInput) could not convert to []interface{}")
					return
				}

				if len(sourceArray) == 1 {
					inputParameterPtr.Source = sourceArray[0]
				}

			}

		}

	default:

		//fmt.Println("---")
		//spew.Dump(original)
		//fmt.Println("---")
		//panic("can not be parsed")

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

// CreateWorkflowStepInputArray _
func CreateWorkflowStepInputArray(original interface{}, context *WorkflowContext) (arrayPtr *[]WorkflowStepInput, err error) {

	array := []WorkflowStepInput{}
	arrayPtr = &array

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	switch original.(type) {
	case map[string]interface{}:
		originalMap := original.(map[string]interface{})

		for k, v := range originalMap {
			//v is an input

			inputParameter, xerr := NewWorkflowStepInput(v, context)
			if xerr != nil {
				err = fmt.Errorf("(CreateWorkflowStepInputArray) (map) NewWorkflowStepInput returns: %s", xerr.Error())
				return
			}

			if inputParameter.ID == "" {
				inputParameter.ID = k
			}

			array = append(array, *inputParameter)
			//fmt.Printf("D")
		}
		return

	case []interface{}:

		for _, v := range original.([]interface{}) {
			//v is an input

			inputParameter, xerr := NewWorkflowStepInput(v, context)
			if xerr != nil {
				err = fmt.Errorf("(CreateWorkflowStepInputArray) (array) NewWorkflowStepInput returns: %s", xerr.Error())
				return
			}

			array = append(array, *inputParameter)
			//fmt.Printf("D")
		}
		return

	default:
		err = fmt.Errorf("(CreateWorkflowStepInputArray) type unknown: %s", reflect.TypeOf(original))

	}
	//spew.Dump(new_array)
	return
}
