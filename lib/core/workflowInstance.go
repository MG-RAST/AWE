package core

import (
	"fmt"
	"reflect"

	"github.com/MG-RAST/AWE/lib/acl"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
)

// WI_STAT_INIT
// state on creation of object
// add to job -> WI_STAT_PENDING

// WI_STAT_PENDING
// Unevaluated workflow_instance has steps, but not tasks or subworkflows yet
// is officially part of job
// Once workflow_instance is deemed ready -> WI_STAT_READY
// it is ready when it has input

// WI_STAT_QUEUED
// tasks have been created and added to TaskMap

// WI_STAT_READY
// has active tasks and subworkflows
// once all tasks and subworkflows are completed -> WI_STAT_COMPLETED

// WI_STAT_COMPLETED
// completed.

const (
	WI_STAT_INIT    = "init"    // initial state on creation
	WI_STAT_PENDING = "pending" // wants to be enqueued but may have unresolved dependencies
	WI_STAT_READY   = "ready"   // a task ready to be enqueued/evaluated (tasks can be enqueued)
	//TASK_STAT_PENDING          = "pending"     // a task that wants to be enqueued
	//TASK_STAT_READY            = "ready"       // a task ready to be enqueued
	WI_STAT_QUEUED = "queued" // tasks have been created
	//WI_STAT_INPROGRESS = "in-progress" // a first workunit has been checkout (this does not guarantee a workunit is running right now)
	//WI_STAT_SUSPEND          = "suspend"
	//TASK_STAT_FAILED           = "failed"
	//TASK_STAT_FAILED_PERMANENT = "failed-permanent" // on exit code 42
	WI_STAT_COMPLETED = "completed"
)

// Object for each subworkflow
type WorkflowInstance struct {
	RWMutex `bson:"-" json:"-" mapstructure:"-"`
	LocalId string `bson:"local_id" json:"id" mapstructure:"local_id"`
	//_Id                 string           `bson:"_id" json:"_id" mapstructure:"_id"` // unique identifier for mongo, includes jobid !
	JobId               string           `bson:"job_id" json:"job_id" mapstructure:"job_id"`
	Acl                 *acl.Acl         `bson:"acl" json:"-"`
	State               string           `bson:"state" json:"state" mapstructure:"state"`                                           // this is unique identifier for the workflow instance
	Workflow_Definition string           `bson:"workflow_definition" json:"workflow_definition" mapstructure:"workflow_definition"` // name of the workflow this instance is derived from
	Inputs              cwl.Job_document `bson:"inputs" json:"inputs" mapstructure:"inputs"`
	Outputs             cwl.Job_document `bson:"outputs" json:"outputs" mapstructure:"outputs"`
	Tasks               []*Task          `bson:"tasks" json:"tasks" mapstructure:"tasks"`
	RemainTasks         int              `bson:"remaintasks" json:"remaintasks" mapstructure:"remaintasks"`
	TotalTasks          int              `bson:"totaltasks" json:"totaltasks" mapstructure:"totaltasks"`
}

func NewWorkflowInstance(local_id string, jobid string, workflow_definition string, inputs cwl.Job_document, job *Job) (wi *WorkflowInstance, err error) {

	if jobid == "" {
		err = fmt.Errorf("(NewWorkflowInstance) jobid == \"\"")
		return
	}

	logger.Debug(3, "(NewWorkflowInstance) _local_id=%s%s, workflow_definition=%s", jobid, local_id, workflow_definition)

	if job == nil {
		err = fmt.Errorf("(NewWorkflowInstance) job==nil ")
		return
	}

	if inputs == nil {
		err = fmt.Errorf("(NewWorkflowInstance) inputs == nil ")
		return
	}

	if len(inputs) == 0 {
		err = fmt.Errorf("(NewWorkflowInstance) len(inputs) == 0 ")
		return
	}

	wi = &WorkflowInstance{LocalId: local_id, JobId: jobid, Workflow_Definition: workflow_definition, Inputs: inputs}
	wi.State = WI_STAT_INIT

	_, err = wi.Init(job)
	if err != nil {
		err = fmt.Errorf("(NewWorkflowInstance) wi.Init returned: %s", err.Error())
		return
	}

	if wi.Acl == nil {
		err = fmt.Errorf("(NewWorkflowInstance) wi.Acl == nil , init failed ? ")
		return
	}

	return
}

func NewWorkflowInstanceFromInterface(original interface{}, job *Job, context *cwl.WorkflowContext) (wi *WorkflowInstance, err error) {
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

		// remaintasks_if, has_remaintasks := original_map["remaintasks"]
		// if !has_remaintasks {
		// 	err = fmt.Errorf("(NewWorkflowInstanceFromInterface) remaintasks is missing")
		// 	return
		// }

		// var remaintasks_int int
		// remaintasks_int, ok = remaintasks_if.(int)
		// if !ok {

		// 	var remaintasks_float64 float64
		// 	remaintasks_float64, ok = remaintasks_if.(float64)
		// 	if ok {
		// 		remaintasks_int = int(remaintasks_float64)

		// 	} else {
		// 		err = fmt.Errorf("(NewWorkflowInstanceFromInterface) remaintasks is not int (%s)", reflect.TypeOf(remaintasks_if))
		// 		return
		// 	}
		// }

		// wi.RemainTasks = remaintasks_int

		// id_if, has_id := original_map["id"]
		// if !has_id {
		// 	err = fmt.Errorf("(NewWorkflowInstanceFromInterface) id is missing")
		// 	return
		// }

		// var id_str string
		// id_str, ok = id_if.(string)
		// if !ok {
		// 	err = fmt.Errorf("(NewWorkflowInstanceFromInterface) id is not string")
		// 	return
		// }
		// if id_str == "" {
		// 	err = fmt.Errorf("(NewWorkflowInstanceFromInterface) id string is empty")
		// 	return
		// }

		// wi.Id = id_str

		// wd_if, has_wd := original_map["workflow_definition"]
		// if !has_wd {
		// 	err = fmt.Errorf("(NewWorkflowInstanceFromInterface) workflow_definition is missing")
		// 	return
		// }

		// var wd_str string
		// wd_str, ok = wd_if.(string)
		// if !ok {
		// 	err = fmt.Errorf("(NewWorkflowInstanceFromInterface) workflow_definition is not string")
		// 	return
		// }
		// if wd_str == "" {
		// 	spew.Dump(original)
		// 	panic("done")
		// 	err = fmt.Errorf("(NewWorkflowInstanceFromInterface) workflow_definition string is empty")
		// 	return
		// }

		//wi.Workflow_Definition = wd_str

		inputs_if, has_inputs := original_map["inputs"]
		if has_inputs {
			var inputs *cwl.Job_document
			inputs, err = cwl.NewJob_documentFromNamedTypes(inputs_if, context)
			if err != nil {
				err = fmt.Errorf("(NewWorkflowInstanceFromInterface) (for inputs) NewJob_document returned: %s", err.Error())
				return
			}

			original_map["inputs"] = *inputs
		}

		outputs_if, has_outputs := original_map["outputs"]
		if has_outputs {
			var outputs *cwl.Job_document
			outputs, err = cwl.NewJob_documentFromNamedTypes(outputs_if, context)
			if err != nil {
				err = fmt.Errorf("(NewWorkflowInstanceFromInterface) (for outputs) NewJob_document returned: %s", err.Error())
				return
			}

			original_map["outputs"] = *outputs

		}

		tasks_if, has_tasks := original_map["tasks"]
		if has_tasks {
			var tasks []*Task
			tasks, err = NewTasksFromInterface(tasks_if, context)
			if err != nil {
				err = fmt.Errorf("(NewWorkflowInstanceFromInterface) (for outputs) NewTasksFromInterface returned: %s", err.Error())
				return
			}

			original_map["tasks"] = tasks

		}

		wi = &WorkflowInstance{}

		err = mapstructure.Decode(original_map, wi)
		if err != nil {
			fmt.Println("original_map:")
			spew.Dump(original_map)
			err = fmt.Errorf("(NewWorkflowInstanceFromInterface) mapstructure.Decode returned: %s", err.Error())
			return
		}

		_, err = wi.Init(job)
		if err != nil {
			err = fmt.Errorf("(NewWorkflowInstanceFromInterface) wi.Init returned: %s", err.Error())
			return
		}

	case *WorkflowInstance:

		var ok bool
		wi, ok = original.(*WorkflowInstance)
		if !ok {
			err = fmt.Errorf("(NewWorkflowInstanceFromInterface) type assertion problem")
			return
		}

		_, err = wi.Init(job)
		if err != nil {
			err = fmt.Errorf("(NewWorkflowInstanceFromInterface) wi.Init returned: %s", err.Error())
			return
		}

	default:
		err = fmt.Errorf("(NewWorkflowInstanceFromInterface) type unknown, %s", reflect.TypeOf(original))
		return
	}
	return

}

func NewWorkflowInstanceArrayFromInterface(original []interface{}, job *Job, context *cwl.WorkflowContext) (wis []*WorkflowInstance, err error) {

	wis = []*WorkflowInstance{}

	for i, _ := range original {
		var wi *WorkflowInstance
		wi, err = NewWorkflowInstanceFromInterface(original[i], job, context)
		if err != nil {
			err = fmt.Errorf("(NewWorkflowInstanceArrayFromInterface) NewWorkflowInstanceFromInterface returned: %s", err.Error())
			return
		}
		wis = append(wis, wi)
	}

	return
}

func (wi *WorkflowInstance) AddTask(task *Task, db_sync string, write_lock bool) (err error) {
	if write_lock {
		err = wi.LockNamed("WorkflowInstance/AddTask")
		if err != nil {
			err = fmt.Errorf("(AddTask) wi.LockNamed returned: %s", err.Error())
			return
		}
		defer wi.Unlock()
	}

	if task.WorkflowInstanceId == "" {
		err = fmt.Errorf("(AddTask) task.WorkflowInstanceId empty")
		return
	}

	wi.Tasks = append(wi.Tasks, task)

	wi.RemainTasks += 1
	wi.TotalTasks = len(wi.Tasks)

	if db_sync == "db_sync_yes" {
		err = wi.Save(false)
		if err != nil {
			err = fmt.Errorf("(WorkflowInstance/AddTask) wi.Save returned: %s", err.Error())
			return
		}
	}
	return
}

func (wi *WorkflowInstance) SetState(state string, db_sync string, write_lock bool) (err error) {
	if write_lock {
		err = wi.LockNamed("WorkflowInstance/SetState")
		if err != nil {
			err = fmt.Errorf("(WorkflowInstance/SetState) wi.LockNamed returned: %s", err.Error())
			return
		}
		defer wi.Unlock()
	}
	wi.State = state

	if db_sync == "db_sync_yes" {
		wi.Save(false)
	}
	return
}

// db_sync is a string because a bool would be misunderstood as a lock indicator ("db_sync_no", db_sync_yes
func (wi *WorkflowInstance) SetTasks(tasks []*Task, db_sync string) (err error) {
	err = wi.LockNamed("WorkflowInstance/SetTasks")
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/SetTasks) wi.LockNamed returned: %s", err.Error())
		return
	}
	defer wi.Unlock()

	wi.Tasks = tasks
	wi.RemainTasks = len(tasks)
	wi.TotalTasks = len(tasks)
	if db_sync == "db_sync_yes" {
		err = wi.Save(false)
		if err != nil {
			return
		}
	}
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

func (wi *WorkflowInstance) TaskCount() (count int) {
	lock, err := wi.RLockNamed("TaskCount")
	if err != nil {
		return
	}
	defer wi.RUnlockNamed(lock)
	count = 0

	if wi.Tasks == nil {
		return
	}
	count = len(wi.Tasks)

	return
}

func (wi *WorkflowInstance) GetId(read_lock bool) (id string, err error) {
	if read_lock {
		var lock ReadLock
		lock, err = wi.RLockNamed("GetId")
		if err != nil {
			err = fmt.Errorf("(WorkflowInstance/GetId) RLockNamed returned: %s", err.Error())
			return
		}
		defer wi.RUnlockNamed(lock)
	}
	id = wi.JobId + wi.LocalId

	return
}

func (wi *WorkflowInstance) GetState(read_lock bool) (state string, err error) {
	if read_lock {
		var lock ReadLock
		lock, err = wi.RLockNamed("GetState")
		if err != nil {
			err = fmt.Errorf("(WorkflowInstance/GetState) RLockNamed returned: %s", err.Error())
			return
		}
		defer wi.RUnlockNamed(lock)
	}
	state = wi.State

	return
}

func (wi *WorkflowInstance) Init(job *Job) (changed bool, err error) {
	changed = false

	wi.RWMutex.Init("WorkflowInstance")

	if wi.Acl == nil {

		if job.Acl.Owner == "" {
			err = fmt.Errorf("(WorkflowInstance/Init) no job.Acl.Owner")
			return
		}

		wi.Acl = &job.Acl
		changed = true
	}

	if wi.Acl == nil {
		err = fmt.Errorf("(WorkflowInstance/Init) still wi.Acl == nil ??? ")
		return
	}

	var t_changed bool
	for i, _ := range wi.Tasks {
		t_changed, err = wi.Tasks[i].Init(job, wi.JobId)
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

func (wi *WorkflowInstance) Save(read_lock bool) (err error) {
	if read_lock {
		var lock ReadLock
		lock, err = wi.RLockNamed("WorkflowInstance/Save")
		if err != nil {
			return
		}
		defer wi.RUnlockNamed(lock)
	}

	if wi.LocalId == "" {
		err = fmt.Errorf("(WorkflowInstance/Save) job id empty")
		return
	}

	if wi.Acl == nil {
		err = fmt.Errorf("(WorkflowInstance/Save) wi.Acl == nil ")
		return
	}

	logger.Debug(1, "(WorkflowInstance/Save)  dbUpsert next: %s", wi.LocalId)
	//spew.Dump(job)

	err = dbUpsert(wi)
	if err != nil {
		spew.Dump(wi)
		err = fmt.Errorf("(WorkflowInstance/Save)  dbUpsert failed (id=%s) error=%s", wi.LocalId, err.Error())
		return
	}
	logger.Debug(1, "(WorkflowInstance/Save)  wi saved: %s", wi.LocalId)
	return
}

func (wi *WorkflowInstance) SetOutputs(outputs cwl.Job_document, context *cwl.WorkflowContext) (err error) {
	err = wi.LockNamed("WorkflowInstance/SetOutputs")
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/SetOutputs) wi.LockNamed returned: %s", err.Error())
		return
	}
	defer wi.Unlock()

	wi.Outputs = outputs
	err = wi.Save(false)
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/SetOutputs)  Save() returned: %s", err.Error())
		return
	}
	return
}

func (wi *WorkflowInstance) GetOutput(name string, read_lock bool) (obj cwl.CWLType, ok bool, err error) {
	if read_lock {
		var lock ReadLock
		lock, err = wi.RLockNamed("GetOutput")
		if err != nil {
			err = fmt.Errorf("(WorkflowInstance/GetOutput) RLockNamed returned: %s", err.Error())
			return
		}
		defer wi.RUnlockNamed(lock)
	}
	ok = true
	for i, _ := range wi.Outputs {
		named_output := wi.Outputs[i]
		if named_output.Id == name {
			obj = named_output.Value
			return
		}

	}
	ok = false
	return
}

func (wi *WorkflowInstance) DecreaseRemainTasks() (remain int, err error) {
	err = wi.LockNamed("WorkflowInstance/DecreaseRemainTasks")
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/DecreaseRemainTasks) wi.LockNamed returned: %s", err.Error())
		return
	}
	defer wi.Unlock()

	if wi.RemainTasks <= 0 {
		err = fmt.Errorf("(WorkflowInstance/DecreaseRemainTasks) RemainTasks is already %d", wi.RemainTasks)
		return
	}

	wi.RemainTasks -= 1

	//err = dbUpdateJobWorkflow_instancesFieldInt(wi.JobId, wi.Id, "remaintasks", wi.RemainTasks)
	err = dbIncrementJobWorkflow_instancesField(wi.JobId, wi.LocalId, "remaintasks", -1) // TODO return correct value for remain
	//err = wi.Save()
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/DecreaseRemainTasks)  Save() returned: %s", err.Error())
		return
	}
	remain = wi.RemainTasks

	return
}
