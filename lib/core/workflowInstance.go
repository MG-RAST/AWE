package core

import (
	"fmt"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	"github.com/davecgh/go-spew/spew"
	"reflect"
)

type WorkflowInstance struct {
	Id          string           `bson:"id" json:"id" mapstructure:"id"`
	Inputs      cwl.Job_document `bson:"inputs" json:"inputs" mapstructure:"inputs"`
	Outputs     cwl.Job_document `bson:"outputs" json:"outputs" mapstructure:"outputs"`
	RemainTasks int              `bson:"remaintasks" json:"remaintasks" mapstructure:"remaintasks"`
}

func NewWorkflowInstanceFromInterface(original interface{}) (wi WorkflowInstance, err error) {
	original, err = cwl.MakeStringMap(original)
	if err != nil {
		return
	}

	switch original.(type) {
	case map[string]interface{}:

		original_map, ok := original.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewWorkflowInstanceFromInterface) not a map: %s", spew.Sdump(original))
			return
		}

		wi = WorkflowInstance{}

		remaintasks_if, has_remaintasks := original_map["remaintasks"]
		if !has_remaintasks {
			err = fmt.Errorf("(NewWorkflowInstanceFromInterface) remaintasks is missing")
			return
		}

		var remaintasks_int int
		remaintasks_int, ok = remaintasks_if.(int)
		if !ok {

			var remaintasks_float64 float64
			remaintasks_float64, ok = remaintasks_if.(float64)
			if ok {
				remaintasks_int = int(remaintasks_float64)

			} else {
				err = fmt.Errorf("(NewWorkflowInstanceFromInterface) remaintasks is not int (%s)", reflect.TypeOf(remaintasks_if))
				return
			}
		}

		wi.RemainTasks = remaintasks_int

		id_if, has_id := original_map["id"]
		if !has_id {
			err = fmt.Errorf("(NewWorkflowInstanceFromInterface) id is missing")
			return
		}

		var id_str string
		id_str, ok = id_if.(string)
		if !ok {
			err = fmt.Errorf("(NewWorkflowInstanceFromInterface) id is not string")
			return
		}
		if id_str == "" {
			err = fmt.Errorf("(NewWorkflowInstanceFromInterface) id string is empty")
			return
		}

		wi.Id = id_str

		inputs_if, has_inputs := original_map["inputs"]
		if !has_inputs {
			err = fmt.Errorf("(NewWorkflowInstanceFromInterface) inputs missing")
			return
		}

		var inputs *cwl.Job_document
		inputs, err = cwl.NewJob_documentFromNamedTypes(inputs_if)
		if err != nil {
			err = fmt.Errorf("(NewWorkflowInstanceFromInterface) (for inputs) NewJob_document returned: %s", err.Error())
			return
		}

		wi.Inputs = *inputs

		outputs_if, has_outputs := original_map["outputs"]
		if has_outputs {
			var outputs *cwl.Job_document
			outputs, err = cwl.NewJob_documentFromNamedTypes(outputs_if)
			if err != nil {
				err = fmt.Errorf("(NewWorkflowInstanceFromInterface) (for outputs) NewJob_document returned: %s", err.Error())
				return
			}

			wi.Outputs = *outputs

		}

	case *WorkflowInstance:

		wi_ptr, ok := original.(*WorkflowInstance)
		if !ok {
			err = fmt.Errorf("(NewWorkflowInstanceFromInterface) type assertion problem")
			return
		}

		wi = *wi_ptr

	default:
		err = fmt.Errorf("(NewWorkflowInstanceFromInterface) type unknown, %s", reflect.TypeOf(original))
		return
	}
	return

}
