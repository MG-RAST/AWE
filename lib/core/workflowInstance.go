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

// NewWorkflowInstance _
func NewWorkflowInstance(localID string, jobid string, workflowDefinition string, job *Job, parentWorkflowInstanceID string) (wi *WorkflowInstance, err error) {

	if jobid == "" {
		err = fmt.Errorf("(NewWorkflowInstance) jobid == \"\"")
		return
	}

	logger.Debug(3, "(NewWorkflowInstance) _local_id=%s%s, workflow_definition=%s", jobid, localID, workflowDefinition)

	if job == nil {
		err = fmt.Errorf("(NewWorkflowInstance) job==nil ")
		return
	}

	wi = &WorkflowInstance{LocalID: localID, JobID: jobid, WorkflowDefinition: workflowDefinition}
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

// NewWorkflowInstanceFromInterface _
func NewWorkflowInstanceFromInterface(original interface{}, job *Job, context *cwl.WorkflowContext, doInit bool) (wi *WorkflowInstance, err error) {
	original, err = cwl.MakeStringMap(original, context)
	if err != nil {
		return
	}

	switch original.(type) {
	case map[string]interface{}:

		originalMap, ok := original.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewWorkflowInstanceFromInterface) not a map: %s", spew.Sdump(original))
			return
		}

		inputsIf, hasInputs := originalMap["inputs"]
		if hasInputs {
			var inputs *cwl.Job_document
			inputs, err = cwl.NewJob_documentFromNamedTypes(inputsIf, context)
			if err != nil {
				err = fmt.Errorf("(NewWorkflowInstanceFromInterface) (for inputs) NewJob_document returned: %s", err.Error())
				return
			}

			originalMap["inputs"] = *inputs
		}

		outputsIf, hasOutputs := originalMap["outputs"]
		if hasOutputs {
			var outputs *cwl.Job_document
			outputs, err = cwl.NewJob_documentFromNamedTypes(outputsIf, context)
			if err != nil {
				err = fmt.Errorf("(NewWorkflowInstanceFromInterface) (for outputs) NewJob_document returned: %s", err.Error())
				return
			}

			originalMap["outputs"] = *outputs

		}

		tasksIf, hasTasks := originalMap["tasks"]
		if hasTasks {
			var tasks []*Task
			tasks, err = NewTasksFromInterface(tasksIf, context)
			if err != nil {
				err = fmt.Errorf("(NewWorkflowInstanceFromInterface) (for outputs) NewTasksFromInterface returned: %s", err.Error())
				return
			}

			originalMap["tasks"] = tasks

		}

		wi = &WorkflowInstance{}

		err = mapstructure.Decode(originalMap, wi)
		if err != nil {
			fmt.Println("original_map:")
			spew.Dump(originalMap)
			err = fmt.Errorf("(NewWorkflowInstanceFromInterface) mapstructure.Decode returned: %s", err.Error())
			return
		}

		if doInit {
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

		if doInit {
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
		for i := range wi.Inputs {
			inpNamed := &wi.Inputs[i]
			inpID := inpNamed.Id
			inpValue := inpNamed.Value

			err = context.Add(inpID, inpValue, "NewWorkflowInstanceFromInterface")
			if err != nil {
				err = fmt.Errorf("(NewWorkflowInstanceFromInterface) context.Add returned: %s", err.Error())
				return
			}
		}
	}
	return

}

// NewWorkflowInstanceArrayFromInterface _
func NewWorkflowInstanceArrayFromInterface(original []interface{}, job *Job, context *cwl.WorkflowContext) (wis []*WorkflowInstance, err error) {

	wis = []*WorkflowInstance{}

	for i := range original {
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

// AddTask db_sync is a string because a bool would be misunderstood as a lock indicator ("db_sync_no", db_sync_yes)
func (wi *WorkflowInstance) AddTask(job *Job, task *Task, dbSync bool, writeLock bool) (err error) {
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
	if dbSync == DbSyncTrue {

		var jobID string
		jobID, err = job.GetId(false)
		if err != nil {
			return
		}
		subworkflowID := task.WorkflowInstanceId

		err = dbPushTask(jobID, subworkflowID, task)
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
// DO NOT CALL directely, use wrapper qm.WISetState()
func (wi *WorkflowInstance) SetState(state string, dbSync bool, writeLock bool) (err error) {

	err = wi.setStateOnly(state, dbSync, writeLock)
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/SetState) setStateOnly returned: %s", err.Error())
		return
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

	newSubworkflowsList := append(wi.Subworkflows, subworkflow)

	jobID := wi.JobID
	subworkflowID := wi.LocalID
	fieldname := "subworkflows"
	updateValue := bson.M{fieldname: newSubworkflowsList}
	err = dbUpdateJobWorkflow_instancesFields(jobID, subworkflowID, updateValue)
	if err != nil {
		err = fmt.Errorf("(AddSubworkflow) (subworkflow_id: %s, fieldname: %s) %s", subworkflowID, fieldname, err.Error())
		return
	}

	wi.Subworkflows = newSubworkflowsList

	_, err = wi.IncrementRemainSteps(1, false)
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/AddSubworkflow) wi.IncrementRemainSteps returned: %s", err.Error())
		return
	}

	return
}

// GetWorkflow _
func (wi *WorkflowInstance) GetWorkflow(context *cwl.WorkflowContext) (workflow *cwl.Workflow, err error) {

	workflowDefStr := wi.WorkflowDefinition

	if context == nil {
		err = fmt.Errorf("(WorkflowInstance/GetWorkflow) context == nil")
		return
	}

	workflow, err = context.GetWorkflow(workflowDefStr)
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
		baseName := path.Base(t.TaskName)
		logger.Debug(3, "(GetTaskByName) task: %s", t.TaskName)
		if baseName == taskName {
			ok = true
			task = t
		}
	}

	return
}

// GetTasks get tasks form from all subworkflows in the job
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

// TaskCount _
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

// GetState _
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

// Init _
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

	var tChanged bool
	for i := range wi.Tasks {
		tChanged, err = wi.Tasks[i].Init(job, wi.JobID)
		if err != nil {
			err = fmt.Errorf("(WorkflowInstance/Init) task.Init returned: %s", err.Error())
			return
		}
		if tChanged {
			changed = true
		}
	}

	return
}

// Save _
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

// SetOutputs _
func (wi *WorkflowInstance) SetOutputs(outputs cwl.Job_document, context *cwl.WorkflowContext, writeLock bool) (err error) {
	if writeLock {
		err = wi.LockNamed("WorkflowInstance/SetOutputs")
		if err != nil {
			err = fmt.Errorf("(WorkflowInstance/SetOutputs) wi.LockNamed returned: %s", err.Error())
			return
		}
		defer wi.Unlock()
	}
	err = dbUpdateJobWorkflow_instancesField(wi.JobID, wi.LocalID, "outputs", outputs)
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/SetOutputs) dbUpdateJobWorkflow_instancesField returned: %s", err.Error())
		return
	}

	err = wi.Save(false)
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/SetOutputs)  Save() returned: %s", err.Error())
		return
	}

	wi.Outputs = outputs

	return
}

// GetOutput _
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

	for i := range wi.Outputs {
		namedOutput := wi.Outputs[i]
		namedOutputBase := path.Base(namedOutput.Id)
		if namedOutputBase == name {
			obj = namedOutput.Value
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

	//var job *Job
	//job, err = wi.GetJob(false)
	//if err != nil {
	//	return
	//}
	//_, err = job.IncrementRemainSteps(-1, "WorkflowInstance/DecreaseRemainSteps")
	//if err != nil {
	//	err = fmt.Errorf("(WorkflowInstance/DecreaseRemainSteps) job.IncrementRemainSteps returned: %s", err.Error())
	//	return
	//}

	return
}

// IncrementRemainSteps _
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

	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/IncrementRemainSteps)  dbIncrementJobWorkflow_instancesField() returned: %s", err.Error())
		return
	}
	wi.RemainSteps += amount

	remain = wi.RemainSteps

	return
}

// GetRemainSteps _
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

// GetParentRaw _
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

// GetParent _
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

		keys := ""
		for key := range job.WorkflowInstancesMap {
			keys += "," + key
		}

		err = fmt.Errorf("(WorkflowInstance/GetParent) job.GetWorkflowInstance did not find workflow_instance %s (only got %s)", parentID, keys)
		return
	}

	return
}

func (wi *WorkflowInstance) GetParentStep_DEPRECATED(readLock bool) (pstep *cwl.WorkflowStep, err error) {

	pstep = wi.ParentStep

	if pstep != nil {
		return
	}

	var parentID string
	parentID, err = wi.GetParentID(false)
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/GetParentStep) GetParentID returned: %s", err.Error())
		return
	}

	var job *Job
	job, err = wi.GetJob(readLock)

	var parentWI *WorkflowInstance
	var ok bool
	parentWI, ok, err = job.GetWorkflowInstance(parentID, true)
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/GetParentStep) job.GetWorkflowInstance returned: %s", err.Error())
		return
	}

	if !ok {
		err = fmt.Errorf("(WorkflowInstance/GetParentStep) job.GetWorkflowInstance did not workflow_instance %s", parentID)
		return
	}

	//parentStep = parent_wi.GetStep()
	parentWorkflow := parentWI.Workflow
	if parentWorkflow == nil {
		err = fmt.Errorf("(WorkflowInstance/GetParentStep) parent_workflow == nil")
		return
	}

	parentIDBase := path.Base(parentID)

	pstep, err = parentWorkflow.GetStep(parentIDBase)
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/GetParentStep) parent_workflow.GetStep returned: %s", err.Error())
		return
	}
	return
}

// GetParentID returns relative ID of parent workflow
func (wi *WorkflowInstance) GetParentID(readLock bool) (parentID string, err error) {
	if readLock {
		var lock rwmutex.ReadLock
		lock, err = wi.RLockNamed("GetParentID")
		if err != nil {
			err = fmt.Errorf("(WorkflowInstance/GetParentID) RLockNamed returned: %s", err.Error())
			return
		}
		defer wi.RUnlockNamed(lock)
	}

	logger.Debug(3, "(GetParentID) %s", wi.LocalID)
	parentID = path.Dir(wi.LocalID)
	logger.Debug(3, "(GetParentID) parent_id: %s", parentID)
	if parentID == "." {
		parentID = ""
	}

	return

}

// GetJob _
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

	jobID := wi.JobID

	job, err = GetJob(jobID)
	if err != nil {
		err = fmt.Errorf("(WorkflowInstance/GetJob) GetJob returned: %s", err.Error())
		return
	}

	wi.Job = job

	return

}
