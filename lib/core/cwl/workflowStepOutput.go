package cwl

import (
	"fmt"
	//"github.com/davecgh/go-spew/spew"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

type WorkflowStepOutput struct {
	CWL_object_Impl `yaml:",inline" bson:",inline" json:",inline" mapstructure:",squash"` // provides Is_CWL_object
	Id              string                                                                `yaml:"id" bson:"id" json:"id" mapstructure:"id"`
}

func NewWorkflowStepOutput(original interface{}, context *WorkflowContext) (wso_ptr *WorkflowStepOutput, err error) {

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	var wso WorkflowStepOutput

	switch original.(type) {
	case string:
		original_str := original.(string)
		wso = WorkflowStepOutput{Id: original_str}
		wso_ptr = &wso
		return
	case map[string]interface{}:
		err = mapstructure.Decode(original, &wso)
		if err != nil {
			err = fmt.Errorf("(CreateWorkflowStepOutputArray) %s", err.Error())
			return
		}
		wso_ptr = &wso
		return
	default:
		err = fmt.Errorf("(NewWorkflowStepOutput) could not parse NewWorkflowStepOutput, type unknown %s", reflect.TypeOf(original))

	}

	return
}

func NewWorkflowStepOutputArray(original interface{}, context *WorkflowContext) (new_array []WorkflowStepOutput, err error) {

	switch original.(type) {
	case map[interface{}]interface{}:

		for k, v := range original.(map[interface{}]interface{}) {
			//fmt.Printf("A")

			wso, xerr := NewWorkflowStepOutput(v, context)
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

			wso, xerr := NewWorkflowStepOutput(v, context)
			if xerr != nil {
				err = xerr
				return
			}
			new_array = append(new_array, *wso)

		}
	default:
		err = fmt.Errorf("(NewWorkflowStepOutputArray) type unknown %s", reflect.TypeOf(original))
		return
	} // end switch

	//spew.Dump(new_array)
	return
}
