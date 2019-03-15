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
	WI_STAT_QUEUED  = "queued"  // tasks have been created
	//WI_STAT_INPROGRESS = "in-progress" // a first workunit has been checkout (this does not guarantee a workunit is running right now)
	//WI_STAT_SUSPEND          = "suspend"
	//WI_STAT_FAILED           = "failed"
	//WI_STAT_FAILED_PERMANENT = "failed-permanent" // on exit code 42
	WI_STAT_COMPLETED = "completed"

	WI_STAT_SUSPEND = "suspend"
)

// WorkflowInstance _
type WorkflowInstance struct {
	rwmutex.RWMutex `bson:"-" json:"-" mapstructure:"-"`
	LocalID         string `bson:"local_id" json:"id" mapstructure:"local_id"` // workfow id without job id , mongo uses JobId_LocalId to get a globally unique identifier
	JobID           string `bson:"job_id" json:"job_id" mapstructure:"job_id"`
	//ParentID           string            `bson:"parent_id" json:"parent_id" mapstructure:"parent_id"` // DEPRECATED!? it can be computed from LocalId
	ACL                *acl.Acl          `bson:"acl" json:"-"`
	State              string            `bson:"state" json:"state" mapstructure:"state"`                                           // this is unique identifier for the workflow instance
	WorkflowDefinition string            `bson:"workflow_definition" json:"workflow_definition" mapstructure:"workflow_definition"` // name of the workflow this instance is derived from
	Workflow           *cwl.Workflow     `bson:"-" json:"-" mapstructure:"-"`                                                       // just a cache for the Workflow pointer
	Inputs             cwl.Job_document  `bson:"inputs" json:"inputs" mapstructure:"inputs"`
	Outputs            cwl.Job_document  `bson:"outputs" json:"outputs" mapstructure:"outputs"`
	Tasks              []*Task           `bson:"tasks" json:"tasks" mapstructure:"tasks"`
	RemainSteps        int               `bson:"remainsteps" json:"remainsteps" mapstructure:"remainsteps"`
	TotalTasks         int               `bson:"totaltasks" json:"totaltasks" mapstructure:"totaltasks"`
	Subworkflows       []string          `bson:"subworkflows" json:"subworkflows" mapstructure:"subworkflows"`
	ParentStep         *cwl.WorkflowStep `bson:"-" json:"-" mapstructure:"-"` // cache
	Parent             *WorkflowInstance `bson:"-" json:"-" mapstructure:"-"` // cache for ParentId
	Job                *Job              `bson:"-" json:"-" mapstructure:"-"` // cache
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

	wi = &WorkflowInstance{LocalID: local_id, JobID: jobid, WorkflowDefinition: workflow_definition}
	wi.State = WI_STAT_INIT

	_, err = wi.Init(job)
	if err != nil {
		err = fmt.Errorf("(NewWorkflowInstance) wi.Init returned: %s", err.Error())
		return
	}

	if wi.ACL == nil {
		err = fmt.Errorf("(NewWorkflowInstance) wi.ACL == nil , init failed ? ")
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
func (wi *WorkflowInstance) AddTask(job *Job, task *Task, db_sync bool, writeLock bool) (err error) {
	if writeLock {
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

	logger.Debug(3, "(WorkflowInstance/AddTask) adding task: %s", task.TaskName)

	for _, t := range wi.Tasks {
		if t.TaskName == task.TaskName {
			err = fmt.Errorf("(WorkflowInstance/AddTask) task with same name already in WorkflowInstance (%s)", task.TaskName)
			return
		}
	}

	_, err = wi.IncrementRemainSteps(1, false)
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/AddTask) wi.IncrementRemainSteps returned: %s", err.Error())
		return
	}

	//_, err = job.IncrementRemainSteps(1)
	//if err != nil {
	//	err = fmt.Errorf("(WorkflowInstance/AddTask) job.IncrementRemainSteps returned: %s", err.Error())
	//	return
	//}

	//wi.TotalTasks = len(wi.Tasks)

	wi.Tasks = append(wi.Tasks, task)
	if db_sync == DbSyncTrue {

		var job_id string
		job_id, err = job.GetId(false)
		if err != nil {
			return
		}
		subworkflow_id := task.WorkflowInstanceId

		err = dbPushTask(job_id, subworkflow_id, task)
		if err != nil {
			err = fmt.Errorf("(WorkflowInstance/AddTask) dbPushTask returned: %s", err.Error())
			return
		}
	}

	return
}

func (wi *WorkflowInstance) setStateOnly(state string, dbSync bool, writeLock bool) (err error) {
	if writeLock {
		err = wi.LockNamed("WorkflowInstance/setStateOnly")
		if err != nil {
			err = fmt.Errorf("(WorkflowInstance/setStateOnly) wi.LockNamed returned: %s", err.Error())
			return
		}
		defer wi.Unlock()
	}

	if dbSync == DbSyncTrue {

		jobID := wi.JobID

		//subworkflowID, _ := wi.GetID(false)
		subworkflowID := wi.LocalID

		err = dbUpdateJobWorkflow_instancesFieldString(jobID, subworkflowID, "state", state)
		if err != nil {
			err = fmt.Errorf("(WorkflowInstance/setStateOnly) dbUpdateJobWorkflow_instancesFieldString returned: %s", err.Error())
			return
		}
		//wi.Save(false)
	}

	wi.State = state

	return
}

// SetState will set state and notify parent WorkflowInstance or Job if completed
func (wi *WorkflowInstance) SetState(state string, dbSync bool, writeLock bool) (err error) {

	err = wi.setStateOnly(state, dbSync, writeLock)
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/SetState) setStateOnly returned: %s", err.Error())
		return
	}

	readLock := writeLock

	if state == WI_STAT_COMPLETED {

		// notify parent: either parent workflow_instance _or_ job

		var parentID string
		parentID, err = wi.GetParentID(readLock)
		if err != nil {
			err = fmt.Errorf("(WorkflowInstance/SetState) GetParentID returned: %s", err.Error())
			return
		}

		if parentID == "" {
			// notify job only

			var job *Job
			job, err = wi.GetJob(readLock)
			if err != nil {
				err = fmt.Errorf("(WorkflowInstance/SetState) wi.GetJob() returned: %s", err.Error())
				return
			}

			// update job (only if workflow instance is root)
			var remain int
			remain, err = job.IncrementRemainSteps(-1)
			if err != nil {
				err = fmt.Errorf("(WorkflowInstance/SetState) job.IncrementRemainSteps returned: %s", err.Error())
				return
			}

			if remain == 0 {
				err = job.SetState(JOB_STAT_COMPLETED, nil)
				if err != nil {
					err = fmt.Errorf("(WorkflowInstance/SetState) job.SetState returned: %s", err.Error())
					return
				}
			}
		} else {
			// notify parent only

			var parentWI *WorkflowInstance
			parentWI, err = wi.GetParent(readLock)
			if err != nil {
				err = fmt.Errorf("(WorkflowInstance/SetState) job.GetParent returned: %s", err.Error())
				return
			}
			var parentRemain int
			parentRemain, err = parentWI.IncrementRemainSteps(-1, true)
			if err != nil {
				err = fmt.Errorf("(WorkflowInstance/SetState) job.IncrementRemainSteps returned: %s", err.Error())
				return
			}

			if parentRemain == 0 {
				// recursive call
				err = parentWI.SetState(WI_STAT_COMPLETED, DbSyncTrue, true)
				if err != nil {
					err = fmt.Errorf("(WorkflowInstance/SetState) parentWI.SetState returned: %s", err.Error())
					return
				}
			}

		}
	}

	return
}

// SetSubworkflows _
func (wi *WorkflowInstance) SetSubworkflows(steps []string, writeLock bool) (err error) {
	if writeLock {
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

// AddSubworkflow assumes WorkflowInstance is in mongo already
func (wi *WorkflowInstance) AddSubworkflow(job *Job, subworkflow string, writeLock bool) (err error) {
	if writeLock {
		err = wi.LockNamed("WorkflowInstance/AddSubworkflow")
		if err != nil {
			err = fmt.Errorf("(WorkflowInstance/AddSubworkflow) wi.LockNamed returned: %s", err.Error())
			return
		}
		defer wi.Unlock()
	}

	new_subworkflows_list := append(wi.Subworkflows, subworkflow)

	job_id := wi.JobID
	subworkflow_id := wi.LocalID
	fieldname := "subworkflows"
	update_value := bson.M{fieldname: new_subworkflows_list}
	err = dbUpdateJobWorkflow_instancesFields(job_id, subworkflow_id, update_value)
	if err != nil {
		err = fmt.Errorf("(AddSubworkflow) (subworkflow_id: %s, fieldname: %s) %s", subworkflow_id, fieldname, err.Error())
		return
	}

	wi.Subworkflows = new_subworkflows_list

	_, err = wi.IncrementRemainSteps(1, false)
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/AddSubworkflow) wi.IncrementRemainSteps returned: %s", err.Error())
		return
	}

	return
}

// GetWorkflow _
func (wi *WorkflowInstance) GetWorkflow(context *cwl.WorkflowContext) (workflow *cwl.Workflow, err error) {

	workflow_def_str := wi.WorkflowDefinition

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

// GetTask _
func (wi *WorkflowInstance) GetTask(taskID Task_Unique_Identifier, readLock bool) (task *Task, ok bool, err error) {
	ok = false
	if readLock {
		var lock rwmutex.ReadLock
		lock, err = wi.RLockNamed("WorkflowInstance/GetTask")
		if err != nil {
			return
		}
		defer wi.RUnlockNamed(lock)
	}
	for _, t := range wi.Tasks {
		if t.Task_Unique_Identifier == taskID {
			ok = true
			task = t
		}
	}

	return
}

// GetTaskByName _
func (wi *WorkflowInstance) GetTaskByName(taskName string, readLock bool) (task *Task, ok bool, err error) {
	ok = false
	if readLock {
		var lock rwmutex.ReadLock
		lock, err = wi.RLockNamed("WorkflowInstance/GetTaskByName")
		if err != nil {
			return
		}
		defer wi.RUnlockNamed(lock)
	}
	logger.Debug(3, "(GetTaskByName) search: %s", taskName)
	for _, t := range wi.Tasks {
		base_name := path.Base(t.TaskName)
		logger.Debug(3, "(GetTaskByName) task: %s", t.TaskName)
		if base_name == taskName {
			ok = true
			task = t
		}
	}

	return
}

// get tasks form from all subworkflows in the job
func (wi *WorkflowInstance) GetTasks(readLock bool) (tasks []*Task, err error) {
	tasks = []*Task{}

	if readLock {
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

// GetID includes JobID
func (wi *WorkflowInstance) GetID(readLock bool) (id string, err error) {
	if readLock {
		var lock rwmutex.ReadLock
		lock, err = wi.RLockNamed("GetID")
		if err != nil {
			err = fmt.Errorf("(WorkflowInstance/GetID) RLockNamed returned: %s", err.Error())
			return
		}
		defer wi.RUnlockNamed(lock)
	}
	id = wi.JobID + "_" + wi.LocalID

	return
}

func (wi *WorkflowInstance) GetState(readLock bool) (state string, err error) {
	if readLock {
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

	if wi.ACL == nil {

		if job.ACL.Owner == "" {
			err = fmt.Errorf("(WorkflowInstance/Init) no job.ACL.Owner")
			return
		}

		wi.ACL = &job.ACL
		changed = true
	}

	if wi.ACL == nil {
		err = fmt.Errorf("(WorkflowInstance/Init) still wi.ACL == nil ??? ")
		return
	}

	var t_changed bool
	for i, _ := range wi.Tasks {
		t_changed, err = wi.Tasks[i].Init(job, wi.JobID)
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

func (wi *WorkflowInstance) Save(readLock bool) (err error) {
	if readLock {
		var lock rwmutex.ReadLock
		lock, err = wi.RLockNamed("WorkflowInstance/Save")
		if err != nil {
			return
		}
		defer wi.RUnlockNamed(lock)
	}

	if wi.LocalID == "" {
		err = fmt.Errorf("(WorkflowInstance/Save) job id empty")
		return
	}

	if wi.ACL == nil {
		err = fmt.Errorf("(WorkflowInstance/Save) wi.ACL == nil ")
		return
	}

	logger.Debug(1, "(WorkflowInstance/Save)  dbUpsert next: %s", wi.LocalID)
	//spew.Dump(job)

	err = dbUpsert(wi)
	if err != nil {
		spew.Dump(wi)
		err = fmt.Errorf("(WorkflowInstance/Save)  dbUpsert failed (id=%s) error=%s", wi.LocalID, err.Error())
		return
	}
	logger.Debug(1, "(WorkflowInstance/Save)  wi saved: %s", wi.LocalID)
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

func (wi *WorkflowInstance) GetOutput(name string, readLock bool) (obj cwl.CWLType, ok bool, err error) {
	if readLock {
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

// also changes remain steps in Job
func (wi *WorkflowInstance) DecreaseRemainSteps_DEPRECATED() (remain int, err error) {
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
	err = dbIncrementJobWorkflow_instancesField(wi.JobID, wi.LocalID, "remainsteps", -1) // TODO return correct value for remain
	//err = wi.Save()
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/DecreaseRemainSteps)  dbIncrementJobWorkflow_instancesField() returned: %s", err.Error())
		return
	}
	remain = wi.RemainSteps

	var job *Job
	job, err = wi.GetJob(false)
	if err != nil {
		return
	}
	_, err = job.IncrementRemainSteps(-1)
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/DecreaseRemainSteps) job.IncrementRemainSteps returned: %s", err.Error())
		return
	}

	return
}

func (wi *WorkflowInstance) IncrementRemainSteps(amount int, writeLock bool) (remain int, err error) {
	if writeLock {
		err = wi.LockNamed("WorkflowInstance/IncrementRemainSteps")
		if err != nil {
			err = fmt.Errorf("(WorkflowInstance/IncrementRemainSteps) wi.LockNamed returned: %s", err.Error())
			return
		}
		defer wi.Unlock()
	}

	err = dbIncrementJobWorkflow_instancesField(wi.JobID, wi.LocalID, "remainsteps", amount) // TODO return correct value for remain
	//err = wi.Save()
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/IncrementRemainSteps)  dbIncrementJobWorkflow_instancesField() returned: %s", err.Error())
		return
	}
	wi.RemainSteps += amount

	remain = wi.RemainSteps

	var job *Job
	job, err = wi.GetJob(false)
	if err != nil {
		return
	}
	_, err = job.IncrementRemainSteps(amount)
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/IncrementRemainSteps) job.IncrementRemainSteps returned: %s", err.Error())
		return
	}

	return
}

func (wi *WorkflowInstance) GetRemainSteps(readLock bool) (remain int, err error) {
	if readLock {
		var lock rwmutex.ReadLock
		lock, err = wi.RLockNamed("GetRemainSteps")
		if err != nil {
			err = fmt.Errorf("(WorkflowInstance/GetRemainSteps) RLockNamed returned: %s", err.Error())
			return
		}
		defer wi.RUnlockNamed(lock)
	}

	remain = wi.RemainSteps

	return
}

func (wi *WorkflowInstance) GetParentStep_cached_DEPRECATED() (pstep *cwl.WorkflowStep, err error) {
	var lock rwmutex.ReadLock
	lock, err = wi.RLockNamed("GetParentStep_cached")
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/GetParentStep_cached) RLockNamed returned: %s", err.Error())
		return
	}
	defer wi.RUnlockNamed(lock)

	pstep = wi.ParentStep

	return
}

func (wi *WorkflowInstance) GetParentRaw(readLock bool) (parent *WorkflowInstance, err error) {
	if readLock {
		var lock rwmutex.ReadLock
		lock, err = wi.RLockNamed("GetParentRaw")
		if err != nil {
			err = fmt.Errorf("(WorkflowInstance/GetParentRaw) RLockNamed returned: %s", err.Error())
			return
		}
		defer wi.RUnlockNamed(lock)
	}
	parent = wi.Parent
	return
}

func (wi *WorkflowInstance) GetParent(readLock bool) (parent *WorkflowInstance, err error) {

	parent, err = wi.GetParentRaw(readLock)
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/GetParent) wi.GetParent returned: %s", err.Error())
		return
	}

	if parent != nil {
		return
	}

	var parentID string
	parentID, err = wi.GetParentID(readLock)
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/GetParent) wi.GetParentID returned: %s", err.Error())
		return
	}

	if parentID == "" {
		err = fmt.Errorf("(WorkflowInstance/GetParent) no parent")
		return
	}

	var job *Job
	job, err = wi.GetJob(readLock)
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/GetParent) wi.GetJob returned: %s", err.Error())
		return
	}

	var ok bool
	parent, ok, err = job.GetWorkflowInstance(parentID, true)
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/GetParent) job.GetWorkflowInstance returned: %s", err.Error())
		return
	}

	if !ok {
		err = fmt.Errorf("(WorkflowInstance/GetParent) job.GetWorkflowInstance did not find workflow_instance %s", parentID)
		return
	}

	return
}

func (wi *WorkflowInstance) GetParentStep_DEPRECATED(readLock bool) (pstep *cwl.WorkflowStep, err error) {

	pstep = wi.ParentStep

	if pstep != nil {
		return
	}

	var parent_id string
	parent_id, err = wi.GetParentID(false)
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/GetParentStep) GetParentID returned: %s", err.Error())
		return
	}

	var job *Job
	job, err = wi.GetJob(readLock)

	var parent_wi *WorkflowInstance
	var ok bool
	parent_wi, ok, err = job.GetWorkflowInstance(parent_id, true)
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/GetParentStep) job.GetWorkflowInstance returned: %s", err.Error())
		return
	}

	if !ok {
		err = fmt.Errorf("(WorkflowInstance/GetParentStep) job.GetWorkflowInstance did not workflow_instance %s", parent_id)
		return
	}

	//parentStep = parent_wi.GetStep()
	parent_workflow := parent_wi.Workflow
	if parent_workflow == nil {
		err = fmt.Errorf("(WorkflowInstance/GetParentStep) parent_workflow == nil")
		return
	}

	parent_id_base := path.Base(parent_id)

	pstep, err = parent_workflow.GetStep(parent_id_base)
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/GetParentStep) parent_workflow.GetStep returned: %s", err.Error())
		return
	}
	return
}

// GetParentID returns relative ID of parent workflow
func (wi *WorkflowInstance) GetParentID(readLock bool) (parent_id string, err error) {
	if readLock {
		var lock rwmutex.ReadLock
		lock, err = wi.RLockNamed("GetParentID")
		if err != nil {
			err = fmt.Errorf("(WorkflowInstance/GetParentID) RLockNamed returned: %s", err.Error())
			return
		}
		defer wi.RUnlockNamed(lock)
	}

	parent_id = path.Base(wi.LocalID)

	return

}

func (wi *WorkflowInstance) GetJob(readLock bool) (job *Job, err error) {
	if readLock {
		var lock rwmutex.ReadLock
		lock, err = wi.RLockNamed("GetJob")
		if err != nil {
			err = fmt.Errorf("(WorkflowInstance/GetJob) RLockNamed returned: %s", err.Error())
			return
		}
		defer wi.RUnlockNamed(lock)
	}

	if wi.Job != nil {
		job = wi.Job
		return
	}

	job_id := wi.JobID

	job, err = GetJob(job_id)
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/GetJob) GetJob returned: %s", err.Error())
		return
	}

	wi.Job = job

	return

}
