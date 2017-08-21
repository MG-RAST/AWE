package cwl

import (
	"fmt"
	//"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
)

type WorkflowStepOutput struct {
	Id string `yaml:"id" bson:"id" json:"id"`
}

func NewWorkflowStepOutput(original interface{}) (wso_ptr *WorkflowStepOutput, err error) {

	var wso WorkflowStepOutput
	err = mapstructure.Decode(original, &wso)
	if err != nil {
		err = fmt.Errorf("(CreateWorkflowStepOutputArray) %s", err.Error())
		return
	}
	return
}

func CreateWorkflowStepOutputArray(original interface{}) (new_array []WorkflowStepOutput, err error) {

	switch original.(type) {
	case map[interface{}]interface{}:

		for k, v := range original.(map[interface{}]interface{}) {
			//fmt.Printf("A")

			wso, xerr := NewWorkflowStepOutput(v)
			//var output_parameter WorkflowStepOutput
			//err = mapstructure.Decode(v, &output_parameter)
			if xerr != nil {
				err = fmt.Errorf("(CreateWorkflowStepOutputArray) %s", xerr.Error())
				return
			}

			wso.Id = k.(string)
			//fmt.Printf("C")
			new_array = append(new_array, *wso)
			//fmt.Printf("D")

		}

	case []interface{}:
		for _, v := range original.([]interface{}) {

			switch v.(type) {
			case string:
				output_parameter := WorkflowStepOutput{Id: v.(string)}
				new_array = append(new_array, output_parameter)
			default:
				wso, ok := v.(WorkflowStepOutput)
				if !ok {
					// TODO some ERROR
				}
				new_array = append(new_array, wso)
			}

		}

	} // end switch

	//spew.Dump(new_array)
	return
}
