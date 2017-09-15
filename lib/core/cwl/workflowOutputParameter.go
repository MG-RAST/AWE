package cwl

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
	"reflect"
)

type WorkflowOutputParameter struct {
	Id             string               `yaml:"id,omitempty" bson:"id,omitempty" json:"id,omitempty"`
	Label          string               `yaml:"label,omitempty" bson:"label,omitempty" json:"label,omitempty"`
	SecondaryFiles []Expression         `yaml:"secondaryFiles,omitempty" bson:"secondaryFiles,omitempty" json:"secondaryFiles,omitempty"` // TODO string | Expression | array<string | Expression>
	Format         []Expression         `yaml:"format,omitempty" bson:"format,omitempty" json:"format,omitempty"`
	Streamable     bool                 `yaml:"streamable,omitempty" bson:"streamable,omitempty" json:"streamable,omitempty"`
	Doc            string               `yaml:"doc,omitempty" bson:"doc,omitempty" json:"doc,omitempty"`
	OutputBinding  CommandOutputBinding `yaml:"outputBinding,omitempty" bson:"outputBinding,omitempty" json:"outputBinding,omitempty"` //TODO
	OutputSource   []string             `yaml:"outputSource,omitempty" bson:"outputSource,omitempty" json:"outputSource,omitempty"`
	LinkMerge      LinkMergeMethod      `yaml:"linkMerge,omitempty" bson:"linkMerge,omitempty" json:"linkMerge,omitempty"`
	Type           []interface{}        `yaml:"type,omitempty" bson:"type,omitempty" json:"type,omitempty"` //WorkflowOutputParameterType TODO CWLType | OutputRecordSchema | OutputEnumSchema | OutputArraySchema | string | array<CWLType | OutputRecordSchema | OutputEnumSchema | OutputArraySchema | string>
}

func NewWorkflowOutputParameter(original interface{}) (wop *WorkflowOutputParameter, err error) {
	var output_parameter WorkflowOutputParameter

	original, err = makeStringMap(original)
	if err != nil {
		return
	}

	switch original.(type) {

	case map[string]interface{}:
		original_map, ok := original.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewWorkflowOutputParameter) type switch error %s", err.Error())
			return
		}

		outputSource, ok := original_map["outputSource"]
		if ok {
			outputSource_str, ok := outputSource.(string)
			if ok {
				original_map["outputSource"] = []string{outputSource_str}
			}
		}

		wop_type, ok := original_map["type"]
		if ok {

			wop_type_array, xerr := NewWorkflowOutputParameterTypeArray(wop_type)
			if xerr != nil {
				err = fmt.Errorf("from NewWorkflowOutputParameterTypeArray: %s", xerr.Error())
				return
			}
			fmt.Println("wop_type_array: \n")
			fmt.Println(reflect.TypeOf(wop_type_array))

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

// WorkflowOutputParameter
func NewWorkflowOutputParameterArray(original interface{}) (new_array_ptr *[]WorkflowOutputParameter, err error) {

	new_array := []WorkflowOutputParameter{}
	switch original.(type) {
	case map[interface{}]interface{}:
		for k, v := range original.(map[interface{}]interface{}) {
			//fmt.Printf("A")

			output_parameter, xerr := NewWorkflowOutputParameter(v)
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

			output_parameter, xerr := NewWorkflowOutputParameter(v)
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
		err = fmt.Errorf("(NewWorkflowOutputParameterArray) type %s unknown", reflect.TypeOf(original))
	}
	//spew.Dump(new_array)
	return
}
