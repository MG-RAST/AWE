package core

import (
	"fmt"
	"path"
	"reflect"

	"github.com/MG-RAST/AWE/lib/acl"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/rwmutex"
	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/mgo.v2/bson"
)

// WI_STAT_INIT
// state on creation of object
// add to job -> WI_STAT_PENDING

// WI_STAT_PENDING
// Unevaluated workflow_instance has steps, but not tasks or subworkflows yet
// is officially part of job
// Once workflow_instance is deemed ready -> WI_STAT_READY
// it is ready when it has input

// WI_STAT_READY
// has inputs !

// WI_STAT_QUEUED
// tasks have been created and added to TaskMap

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
	rwmutex.RWMutex `bson:"-" json:"-" mapstructure:"-"`
	LocalId         string `bson:"local_id" json:"id" mapstructure:"local_id"`
	//_Id                 string           `bson:"_id" json:"_id" mapstructure:"_id"` // unique identifier for mongo, includes jobid !
	JobId               string            `bson:"job_id" json:"job_id" mapstructure:"job_id"`
	ParentId            string            `bson:"parent_id" json:"parent_id" mapstructure:"parent_id"` // point to workflow_parent parent, this is not trivial duer to embedded workflows
	Acl                 *acl.Acl          `bson:"acl" json:"-"`
	State               string            `bson:"state" json:"state" mapstructure:"state"`                                           // this is unique identifier for the workflow instance
	Workflow_Definition string            `bson:"workflow_definition" json:"workflow_definition" mapstructure:"workflow_definition"` // name of the workflow this instance is derived from
	Workflow            *cwl.Workflow     `bson:"-" json:"-" mapstructure:"-"`                                                       // just a cache for the Workflow pointer
	Inputs              cwl.Job_document  `bson:"inputs" json:"inputs" mapstructure:"inputs"`
	Outputs             cwl.Job_document  `bson:"outputs" json:"outputs" mapstructure:"outputs"`
	Tasks               []*Task           `bson:"tasks" json:"tasks" mapstructure:"tasks"`
	RemainSteps         int               `bson:"remainsteps" json:"remainsteps" mapstructure:"remainsteps"`
	TotalTasks          int               `bson:"totaltasks" json:"totaltasks" mapstructure:"totaltasks"`
	Subworkflows        []string          `bson:"subworkflows" json:"subworkflows" mapstructure:"subworkflows"`
	ParentStep          *cwl.WorkflowStep `bson:"-" json:"-" mapstructure:"-"`
	//Created_by          string            `bson:"created_by" json:"created_by" mapstructure:"created_by"`
}

func NewWorkflowInstance(local_id string, jobid string, workflow_definition string, job *Job, parent_workflow_instance_id string) (wi *WorkflowInstance, err error) {

	if jobid == "" {
		err = fmt.Errorf("(NewWorkflowInstance) jobid == \"\"")
		return
	}

	logger.Debug(3, "(NewWorkflowInstance) _local_id=%s%s, workflow_definition=%s", jobid, local_id, workflow_definition)

	if job == nil {
		err = fmt.Errorf("(NewWorkflowInstance) job==nil ")
		return
	}

	wi = &WorkflowInstance{LocalId: local_id, JobId: jobid, Workflow_Definition: workflow_definition, ParentId: parent_workflow_instance_id}
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

func NewWorkflowInstanceFromInterface(original interface{}, job *Job, context *cwl.WorkflowContext, do_init bool) (wi *WorkflowInstance, err error) {
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

		if do_init {
			_, err = wi.Init(job)
			if err != nil {
				err = fmt.Errorf("(NewWorkflowInstanceFromInterface) wi.Init returned: %s", err.Error())
				return
			}
		}
	case *WorkflowInstance:

		var ok bool
		wi, ok = original.(*WorkflowInstance)
		if !ok {
			err = fmt.Errorf("(NewWorkflowInstanceFromInterface) type assertion problem")
			return
		}

		if do_init {
			_, err = wi.Init(job)
			if err != nil {
				err = fmt.Errorf("(NewWorkflowInstanceFromInterface) wi.Init returned: %s", err.Error())
				return
			}
		}
	default:
		err = fmt.Errorf("(NewWorkflowInstanceFromInterface) type unknown, %s", reflect.TypeOf(original))
		return
	}

	if context != nil {
		for i, _ := range wi.Inputs {
			inp_named := &wi.Inputs[i]
			inp_id := inp_named.Id
			inp_value := inp_named.Value

			err = context.Add(inp_id, inp_value, "NewWorkflowInstanceFromInterface")
			if err != nil {
				err = fmt.Errorf("(NewWorkflowInstanceFromInterface) context.Add returned: %s", err.Error())
				return
			}
		}
	}
	return

}

func NewWorkflowInstanceArrayFromInterface(original []interface{}, job *Job, context *cwl.WorkflowContext) (wis []*WorkflowInstance, err error) {

	wis = []*WorkflowInstance{}

	for i, _ := range original {
		var wi *WorkflowInstance
		wi, err = NewWorkflowInstanceFromInterface(original[i], job, context, true)
		if err != nil {
			err = fmt.Errorf("(NewWorkflowInstanceArrayFromInterface) NewWorkflowInstanceFromInterface returned: %s", err.Error())
			return
		}
		wis = append(wis, wi)
	}

	return
}

// db_sync is a string because a bool would be misunderstood as a lock indicator ("db_sync_no", db_sync_yes)
func (wi *WorkflowInstance) AddTask(job *Job, task *Task, db_sync string, write_lock bool) (err error) {
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

	_, err = wi.IncrementRemainSteps(false)
	if err != nil {
		err = fmt.Errorf("(AddTask) wi.IncrementRemainSteps returned: %s", err.Error())
		return
	}

	err = job.IncrementRemainSteps(1)
	if err != nil {
		err = fmt.Errorf("(AddTask) job.IncrementRemainSteps returned: %s", err.Error())
		return
	}

	//wi.TotalTasks = len(wi.Tasks)

	wi.Tasks = append(wi.Tasks, task)
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

func (wi *WorkflowInstance) SetSubworkflows(steps []string, write_lock bool) (err error) {
	if write_lock {
		err = wi.LockNamed("WorkflowInstance/SetSubworkflows")
		if err != nil {
			err = fmt.Errorf("(WorkflowInstance/SetSubworkflows) wi.LockNamed returned: %s", err.Error())
			return
		}
		defer wi.Unlock()
	}
	wi.Subworkflows = steps

	wi.Save(false)

	return
}

func (wi *WorkflowInstance) AddSubworkflow(job *Job, subworkflow string, write_lock bool) (err error) {
	if write_lock {
		err = wi.LockNamed("WorkflowInstance/AddSubworkflow")
		if err != nil {
			err = fmt.Errorf("(WorkflowInstance/AddSubworkflow) wi.LockNamed returned: %s", err.Error())
			return
		}
		defer wi.Unlock()
	}

	new_subworkflows_list := append(wi.Subworkflows, subworkflow)

	job_id := wi.JobId
	subworkflow_id := wi.LocalId
	fieldname := "subworkflows"
	update_value := bson.M{fieldname: new_subworkflows_list}
	err = dbUpdateJobWorkflow_instancesFields(job_id, subworkflow_id, update_value)
	if err != nil {
		err = fmt.Errorf("(AddSubworkflow) (subworkflow_id: %s, fieldname: %s) %s", subworkflow_id, fieldname, err.Error())
		return
	}

	wi.Subworkflows = new_subworkflows_list

	_, err = wi.IncrementRemainSteps(false)
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/AddSubworkflow) wi.IncrementRemainSteps returned: %s", err.Error())
		return
	}

	err = job.IncrementRemainSteps(1)
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/AddSubworkflow) job.IncrementRemainSteps returned: %s", err.Error())
		return
	}

	return
}

func (wi *WorkflowInstance) GetWorkflow(context *cwl.WorkflowContext) (workflow *cwl.Workflow, err error) {

	workflow_def_str := wi.Workflow_Definition

	if context == nil {
		err = fmt.Errorf("(WorkflowInstance/GetWorkflow) context == nil")
		return
	}

	workflow, err = context.GetWorkflow(workflow_def_str)
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/GetWorkflow) context.GetWorkflow returned: %s", err.Error())
		return
	}

	return
}

func (wi *WorkflowInstance) GetTask(task_id Task_Unique_Identifier, read_lock bool) (task *Task, ok bool, err error) {
	ok = false
	if read_lock {
		var lock rwmutex.ReadLock
		lock, err = wi.RLockNamed("WorkflowInstance/GetTask")
		if err != nil {
			return
		}
		defer wi.RUnlockNamed(lock)
	}
	for _, t := range wi.Tasks {
		if t.Task_Unique_Identifier == task_id {
			ok = true
			task = t
		}
	}

	return
}

// get tasks form from all subworkflows in the job
func (wi *WorkflowInstance) GetTasks(read_lock bool) (tasks []*Task, err error) {
	tasks = []*Task{}

	if read_lock {
		var lock rwmutex.ReadLock
		lock, err = wi.RLockNamed("WorkflowInstance/GetTasks")
		if err != nil {
			return
		}
		defer wi.RUnlockNamed(lock)
	}

	logger.Debug(3, "(GetTasks) is not cwl")
	for _, task := range wi.Tasks {
		tasks = append(tasks, task)
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
		var lock rwmutex.ReadLock
		lock, err = wi.RLockNamed("GetId")
		if err != nil {
			err = fmt.Errorf("(WorkflowInstance/GetId) RLockNamed returned: %s", err.Error())
			return
		}
		defer wi.RUnlockNamed(lock)
	}
	id = wi.JobId + "_" + wi.LocalId

	return
}

func (wi *WorkflowInstance) GetState(read_lock bool) (state string, err error) {
	if read_lock {
		var lock rwmutex.ReadLock
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
		var lock rwmutex.ReadLock
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
		var lock rwmutex.ReadLock
		lock, err = wi.RLockNamed("GetOutput")
		if err != nil {
			err = fmt.Errorf("(WorkflowInstance/GetOutput) RLockNamed returned: %s", err.Error())
			return
		}
		defer wi.RUnlockNamed(lock)
	}
	ok = true

	if wi.Outputs == nil {
		err = fmt.Errorf("(WorkflowInstance/GetOutput) not Outputs")
		return
	}

	for i, _ := range wi.Outputs {
		named_output := wi.Outputs[i]
		named_output_base := path.Base(named_output.Id)
		if named_output_base == name {
			obj = named_output.Value
			return
		}

	}
	ok = false
	return
}

func (wi *WorkflowInstance) DecreaseRemainSteps() (remain int, err error) {
	err = wi.LockNamed("WorkflowInstance/DecreaseRemainSteps")
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/DecreaseRemainSteps) wi.LockNamed returned: %s", err.Error())
		return
	}
	defer wi.Unlock()

	if wi.RemainSteps <= 0 {
		err = fmt.Errorf("(WorkflowInstance/DecreaseRemainSteps) RemainSteps is already %d", wi.RemainSteps)
		return
	}

	wi.RemainSteps -= 1

	//err = dbUpdateJobWorkflow_instancesFieldInt(wi.JobId, wi.Id, "remainsteps", wi.RemainSteps)
	err = dbIncrementJobWorkflow_instancesField(wi.JobId, wi.LocalId, "remainsteps", -1) // TODO return correct value for remain
	//err = wi.Save()
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/DecreaseRemainSteps)  dbIncrementJobWorkflow_instancesField() returned: %s", err.Error())
		return
	}
	remain = wi.RemainSteps

	return
}

func (wi *WorkflowInstance) IncrementRemainSteps(write_lock bool) (remain int, err error) {
	if write_lock {
		err = wi.LockNamed("WorkflowInstance/IncrementRemainSteps")
		if err != nil {
			err = fmt.Errorf("(WorkflowInstance/IncrementRemainSteps) wi.LockNamed returned: %s", err.Error())
			return
		}
		defer wi.Unlock()
	}
	wi.RemainSteps += 1

	err = dbIncrementJobWorkflow_instancesField(wi.JobId, wi.LocalId, "remainsteps", 1) // TODO return correct value for remain
	//err = wi.Save()
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/IncrementRemainSteps)  dbIncrementJobWorkflow_instancesField() returned: %s", err.Error())
		return
	}
	remain = wi.RemainSteps

	return
}
