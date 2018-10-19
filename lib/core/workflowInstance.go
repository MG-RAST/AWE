package core

import (
	"fmt"
	"reflect"

	"github.com/MG-RAST/AWE/lib/core/cwl"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
)

// Object for each subworkflow
type WorkflowInstance struct {
	RWMutex
	Id                  string           `bson:"id" json:"id" mapstructure:"id"`                                                    // this is unique identifier for the workflow instance
	Workflow_Definition string           `bson:"workflow_definition" json:"workflow_definition" mapstructure:"workflow_definition"` // name of the workflow this instance is derived from
	Inputs              cwl.Job_document `bson:"inputs" json:"inputs" mapstructure:"inputs"`
	Outputs             cwl.Job_document `bson:"outputs" json:"outputs" mapstructure:"outputs"`
	Tasks               []*Task          `bson:"tasks" json:"tasks" mapstructure:"tasks"`
	RemainTasks         int              `bson:"remaintasks" json:"remaintasks" mapstructure:"remaintasks"`
}

func NewWorkflowInstance(id string, workflow_definition string, inputs cwl.Job_document, remain_tasks int) *WorkflowInstance {
	return &WorkflowInstance{Id: id, Workflow_Definition: workflow_definition, Inputs: inputs, RemainTasks: remain_tasks}
}

func NewWorkflowInstanceFromInterface(original interface{}, context *cwl.WorkflowContext) (wi WorkflowInstance, err error) {
	original, err = cwl.MakeStringMap(original, context)
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

		wd_if, has_wd := original_map["workflow_definition"]
		if !has_wd {
			err = fmt.Errorf("(NewWorkflowInstanceFromInterface) workflow_definition is missing")
			return
		}

		var wd_str string
		wd_str, ok = wd_if.(string)
		if !ok {
			err = fmt.Errorf("(NewWorkflowInstanceFromInterface) workflow_definition is not string")
			return
		}
		if wd_str == "" {
			spew.Dump(original)
			panic("done")
			err = fmt.Errorf("(NewWorkflowInstanceFromInterface) workflow_definition string is empty")
			return
		}

		wi.Workflow_Definition = wd_str

		inputs_if, has_inputs := original_map["inputs"]
		if !has_inputs {
			err = fmt.Errorf("(NewWorkflowInstanceFromInterface) inputs missing")
			return
		}

		var inputs *cwl.Job_document
		inputs, err = cwl.NewJob_documentFromNamedTypes(inputs_if, context)
		if err != nil {
			err = fmt.Errorf("(NewWorkflowInstanceFromInterface) (for inputs) NewJob_document returned: %s", err.Error())
			return
		}

		wi.Inputs = *inputs

		outputs_if, has_outputs := original_map["outputs"]
		if has_outputs {
			var outputs *cwl.Job_document
			outputs, err = cwl.NewJob_documentFromNamedTypes(outputs_if, context)
			if err != nil {
				err = fmt.Errorf("(NewWorkflowInstanceFromInterface) (for outputs) NewJob_document returned: %s", err.Error())
				return
			}

			wi.Outputs = *outputs

		}

		tasks_if, has_tasks := original_map["tasks"]
		if has_tasks {
			var tasks []*Task
			tasks, err = NewTasksFromInterface(tasks_if, context)
			if err != nil {
				err = fmt.Errorf("(NewWorkflowInstanceFromInterface) (for outputs) NewTasksFromInterface returned: %s", err.Error())
				return
			}

			wi.Tasks = tasks

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

func (wi *WorkflowInstance) AddTask(task *Task) (err error) {
	err = wi.LockNamed("AddTask")
	if err != nil {
		return
	}
	defer wi.Unlock()

	wi.Tasks = append(wi.Tasks, task)
	wi.Save()
	return
}

func (wi *WorkflowInstance) GetTask(task_id Task_Unique_Identifier) (task *Task, ok bool, err error) {
	ok = false

	for _, t := range wi.Tasks {
		if t.Task_Unique_Identifier == task_id {
			ok = true
			task = t
		}
	}

	return
}

func (wi *WorkflowInstance) Init(job *Job) (changed bool, err error) {
	changed = false

	wi.RWMutex.Init("WorkflowInstance")

	if wi.Tasks == nil {
		return
	}

	var t_changed bool
	for i, _ := range wi.Tasks {
		t_changed, err = wi.Tasks[i].Init(job)
		if err != nil {
			err = fmt.Errorf("(WorkflowInstance/Init) task.Init returned: %s", err.Error())
			return
		}
		if t_changed {
			changed = true
		}
	}

	return
}

func (wi *WorkflowInstance) Save() (err error) {
	err = wi.LockNamed("WorkflowInstance/Save")
	if err != nil {
		return
	}
	defer wi.Unlock()

	if wi.Id == "" {
		err = fmt.Errorf("(WorkflowInstance/Save) job id empty")
		return
	}

	logger.Debug(1, "(WorkflowInstance/Save)  dbUpsert next: %s", wi.Id)
	//spew.Dump(job)

	err = dbUpsert(wi)
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/Save)  dbUpsert failed (id=%s) error=%s", wi.Id, err.Error())
		return
	}
	logger.Debug(1, "(WorkflowInstance/Save)  wi saved: %s", wi.Id)
	return
}

func (wi *WorkflowInstance) SetOutputs(outputs cwl.Job_document, context *cwl.WorkflowContext) (err error) {
	err = wi.LockNamed("WorkflowInstance/SetOutputs")
	if err != nil {
		return
	}
	defer wi.Unlock()

	wi.Outputs = outputs
	wi.Save()
	return
}

func (wi *WorkflowInstance) DecreaseRemainTasks() (err error) {
	err = wi.LockNamed("WorkflowInstance/DecreaseRemainTasks")
	if err != nil {
		return
	}
	defer wi.Unlock()

	if wi.RemainTasks <= 0 {
		err = fmt.Errorf("(WorkflowInstance/DecreaseRemainTasks) RemainTasks is already %d", wi.RemainTasks)
		return
	}

	wi.RemainTasks -= 1
	wi.Save()
	return
}
