package cwl

import (
	"fmt"
	cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
	"reflect"
)

type WorkflowOutputParameter struct {
	Id             string                        `yaml:"id" bson:"id" json:"id"`
	Label          string                        `yaml:"label" bson:"label" json:"label"`
	SecondaryFiles []cwl_types.Expression        `yaml:"secondaryFiles" bson:"secondaryFiles" json:"secondaryFiles"` // TODO string | Expression | array<string | Expression>
	Format         []cwl_types.Expression        `yaml:"format" bson:"format" json:"format"`
	Streamable     bool                          `yaml:"streamable" bson:"streamable" json:"streamable"`
	Doc            string                        `yaml:"doc" bson:"doc" json:"doc"`
	OutputBinding  CommandOutputBinding          `yaml:"outputBinding" bson:"outputBinding" json:"outputBinding"` //TODO
	OutputSource   []string                      `yaml:"outputSource" bson:"outputSource" json:"outputSource"`
	LinkMerge      LinkMergeMethod               `yaml:"linkMerge" bson:"linkMerge" json:"linkMerge"`
	Type           []WorkflowOutputParameterType `yaml:"type" bson:"type" json:"type"` // TODO CWLType | OutputRecordSchema | OutputEnumSchema | OutputArraySchema | string | array<CWLType | OutputRecordSchema | OutputEnumSchema | OutputArraySchema | string>
}

func NewWorkflowOutputParameter(original interface{}) (wop *WorkflowOutputParameter, err error) {
	var output_parameter WorkflowOutputParameter

	original_map, ok := original.(map[interface{}]interface{})
	if !ok {
		err = fmt.Errorf("(NewWorkflowOutputParameter) type unknown")
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

		original_map["type"] = *wop_type_array

	}

	err = mapstructure.Decode(original, &output_parameter)
	if err != nil {
		err = fmt.Errorf("(NewWorkflowOutputParameter) decode error: %s", err.Error())
		return
	}
	wop = &output_parameter
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
		err = fmt.Errorf("(NewWorkflowOutputParameterArray) type unknown")
	}
	//spew.Dump(new_array)
	return
}
