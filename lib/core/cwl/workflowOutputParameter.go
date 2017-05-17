package cwl

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
)

type WorkflowOutputParameter struct {
	Id             string                        `yaml:"id"`
	Label          string                        `yaml:"label"`
	SecondaryFiles []Expression                  `yaml:"secondaryFiles"` // TODO string | Expression | array<string | Expression>
	Format         []Expression                  `yaml:"format"`
	Streamable     bool                          `yaml:"streamable"`
	Doc            string                        `yaml:"doc"`
	OutputBinding  CommandOutputBinding          `yaml:"outputBinding"` //TODO
	OutputSource   []string                      `yaml:"outputSource"`
	LinkMerge      LinkMergeMethod               `yaml:"linkMerge"`
	Type           []WorkflowOutputParameterType `yaml:"type"` // TODO CWLType | OutputRecordSchema | OutputEnumSchema | OutputArraySchema | string | array<CWLType | OutputRecordSchema | OutputEnumSchema | OutputArraySchema | string>
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

		original_map["type"], err = NewWorkflowOutputParameterType(wop_type)
		if err != nil {
			return
		}

	}

	err = mapstructure.Decode(original, &output_parameter)
	if err != nil {
		err = fmt.Errorf("(NewWorkflowOutputParameter) %s", err.Error())
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
