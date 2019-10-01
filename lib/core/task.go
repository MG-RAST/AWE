package core

import (
	"errors"
	"fmt"
	"path"
	"reflect"

	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	"github.com/MG-RAST/AWE/lib/logger"
	rwmutex "github.com/MG-RAST/go-rwmutex"
	shock "github.com/MG-RAST/go-shock-client"
	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
)

// hierachy (in ideal case without errors):
// 1. TASK_STAT_INIT
// 2. TASK_STAT_PENDING
// 3. TASK_STAT_READY
// 4. TASK_STAT_QUEUED
// 5. TASK_STAT_INPROGRESS
// 6. TASK_STAT_COMPLETED

// to replaced by ProcessInstance states
const (
	TASK_STAT_INIT             = "init"        // initial state on creation of a task
	TASK_STAT_PENDING          = "pending"     // a task that wants to be enqueued (but dependent tasks are not complete)
	TASK_STAT_READY            = "ready"       // a task ready to be enqueued (all dependent tasks are complete , but workunits habe not yet been created)
	TASK_STAT_QUEUED           = "queued"      // a task for which workunits have been created/queued
	TASK_STAT_INPROGRESS       = "in-progress" // a first workunit has been checkout (this does not guarantee a workunit is running right now)
	TASK_STAT_SUSPEND          = "suspend"
	TASK_STAT_FAILED           = "failed"           // deprecated ?
	TASK_STAT_FAILED_PERMANENT = "failed-permanent" // on exit code 42
	TASK_STAT_COMPLETED        = "completed"
	TASK_STAT_SKIPPED          = "user_skipped" // deprecated
	TASK_STAT_FAIL_SKIP        = "skipped"      // deprecated
	TASK_STAT_PASSED           = "passed"       // deprecated ?
)

var TASK_STATS_RESET = []string{TASK_STAT_QUEUED, TASK_STAT_INPROGRESS, TASK_STAT_SUSPEND}

// Scatter
// A task of type "scatter" generates multiple scatter children.
// List of children for a scatter task are stored in field "ScatterChildren"
// Each Scatter child points to its Scatter parent
// Scatter child outputs do not go into context object, they only go to scatter parent output array !

// TaskRaw _
type TaskRaw struct {
	Task_Unique_Identifier `bson:",inline" mapstructure:",squash"`
	ProcessInstanceBase    `bson:",inline" json:",inline" mapstructure:",squash"`

	ID string `bson:"taskid" json:"taskid" mapstructure:"taskid"` // old-style: jobID_taskname

	Info          *Info                  `bson:"-" json:"-" mapstructure:"-"` // this is just a pointer to the job.Info
	Cmd           *Command               `bson:"cmd" json:"cmd" mapstructure:"cmd"`
	Partition     *PartInfo              `bson:"partinfo" json:"-" mapstructure:"partinfo"`
	DependsOn     []string               `bson:"dependsOn" json:"dependsOn" mapstructure:"dependsOn"` // only needed if dependency cannot be inferred from Input.Origin
	TotalWork     int                    `bson:"totalwork" json:"totalwork" mapstructure:"totalwork"`
	MaxWorkSize   int                    `bson:"maxworksize"   json:"maxworksize" mapstructure:"maxworksize"`
	RemainWork    int                    `bson:"remainwork" json:"remainwork" mapstructure:"remainwork"`
	ResetTask     bool                   `bson:"resettask" json:"-" mapstructure:"resettask"` // trigged by function - resume, recompute, resubmit
	CreatedDate   time.Time              `bson:"createdDate" json:"createddate" mapstructure:"createdDate"`
	StartedDate   time.Time              `bson:"startedDate" json:"starteddate" mapstructure:"startedDate"`
	CompletedDate time.Time              `bson:"completedDate" json:"completeddate" mapstructure:"completedDate"`
	ComputeTime   int                    `bson:"computetime" json:"computetime" mapstructure:"computetime"`
	UserAttr      map[string]interface{} `bson:"userattr" json:"userattr" mapstructure:"userattr"`
	ClientGroups  string                 `bson:"clientgroups" json:"clientgroups" mapstructure:"clientgroups"`
	//WorkflowStep           *cwl.WorkflowStep      `bson:"workflowStep" json:"workflowStep" mapstructure:"workflowStep"`    // CWL-only
	StepOutputInterface    interface{}       `bson:"stepOutput" json:"stepOutput" mapstructure:"stepOutput"`          // CWL-only
	ProcessOutputInterface interface{}       `bson:"processOutput" json:"processOutput" mapstructure:"processOutput"` // CWL-only
	StepInput              *cwl.Job_document `bson:"-" json:"-" mapstructure:"-"`                                     // CWL-only
	StepOutput             *cwl.Job_document `bson:"-" json:"-" mapstructure:"-"`                                     // CWL-only
	ProcessOutput          *cwl.Job_document `bson:"-" json:"-" mapstructure:"-"`                                     // CWL-only
	//Scatter_task        bool                     `bson:"scatter_task" json:"scatter_task" mapstructure:"scatter_task"`                                  // CWL-only, indicates if this is a scatter_task TODO: compare with TaskType ?
	ScatterParent        *Task_Unique_Identifier `bson:"scatter_parent" json:"scatter_parent" mapstructure:"scatter_parent"`                            // CWL-only, points to scatter parent
	ScatterChildren_ptr  []*Task                 `bson:"-" json:"-" mapstructure:"-"`                                                                   // caching only, CWL-only
	Finalizing           bool                    `bson:"-" json:"-" mapstructure:"-"`                                                                   // CWL-only, a lock mechanism for subworkflows and scatter tasks
	CwlVersion           cwl.CWLVersion          `bson:"cwlVersion,omitempty"  mapstructure:"cwlVersion,omitempty" mapstructure:"cwlVersion,omitempty"` // CWL-only
	WorkflowInstanceID   string                  `bson:"workflow_instance_id" json:"workflow_instance_id" mapstructure:"workflow_instance_id"`          // CWL-only
	WorkflowInstanceUUID string                  `bson:"workflow_instance_uuid" json:"workflow_instance_uuid" mapstructure:"workflow_instance_uuid"`    // CWL-only
	job                  *Job                    `bson:"-"  mapstructure:"-"`                                                                           // caching only
	NotReadyReason       string                  `bson:"notReadyReason" json:"notReadyReason" mapstructure:"-"`
	//WorkflowParent      *Task_Unique_Identifier  `bson:"workflow_parent" json:"workflow_parent" mapstructure:"workflow_parent"`                         // CWL-only parent that created subworkflow
}

// Task _
type Task struct {
	TaskRaw `bson:",inline" mapstructure:",squash"`
	Inputs  []*IO `bson:"inputs" json:"inputs" mapstructure:"inputs"`
	Outputs []*IO `bson:"outputs" json:"outputs" mapstructure:"outputs"`
	Predata []*IO `bson:"predata" json:"predata" mapstructure:"predata"`
	Comment string
}

// TaskDep Deprecated JobDep struct uses deprecated TaskDep struct which uses the deprecated IOmap.  Maintained for backwards compatibility.
// Jobs that cannot be parsed into the Job struct, but can be parsed into the JobDep struct will be translated to the new Job struct.
// (=deprecated=)
type TaskDep struct {
	TaskRaw `bson:",inline"`
	Inputs  IOmap `bson:"inputs" json:"inputs"`
	Outputs IOmap `bson:"outputs" json:"outputs"`
	Predata IOmap `bson:"predata" json:"predata"`
}

// TaskLog _
type TaskLog struct {
	ID            string     `bson:"taskid" json:"taskid"`
	State         string     `bson:"state" json:"state"`
	TotalWork     int        `bson:"totalwork" json:"totalwork"`
	CompletedDate time.Time  `bson:"completedDate" json:"completeddate"`
	Workunits     []*WorkLog `bson:"workunits" json:"workunits"`
}

// NewTaskRaw _
func NewTaskRaw(taskID Task_Unique_Identifier, info *Info) (tr *TaskRaw, err error) {

	var taskStr string
	taskStr, err = taskID.String()
	if err != nil {
		err = fmt.Errorf("() task.String returned: %s", err.Error())
		return
	}
	logger.Debug(3, "taskStr: %s", taskStr)
	logger.Debug(3, "taskID.JobId: %s", taskID.JobId)

	logger.Debug(3, "taskID.TaskName: %s", taskID.TaskName)

	tr = &TaskRaw{
		Task_Unique_Identifier: taskID,
		ID:                     taskStr,
		Info:                   info,
		Cmd:                    &Command{},
		Partition:              nil,
		DependsOn:              []string{},
	}

	tr.State = TASK_STAT_INIT

	return
}

// InitRaw _
func (task *TaskRaw) InitRaw(job *Job, jobID string) (changed bool, err error) {
	changed = false

	if len(task.ID) == 0 {
		err = errors.New("(InitRaw) empty taskid")
		return
	}

	//job_id := job.ID

	if jobID == "" {
		err = fmt.Errorf("(InitRaw) jobID empty")
		return
	}

	if task.JobId == "" {
		task.JobId = jobID
		changed = true
	}

	//logger.Debug(3, "task.TaskName A: %s", task.TaskName)
	job_prefix := jobID + "_"
	if len(task.ID) > 0 && (!strings.HasPrefix(task.ID, job_prefix)) {
		task.TaskName = task.ID
		changed = true
		err = fmt.Errorf("(InitRaw) A) broken task? %s", jobID)
		return

	}
	//logger.Debug(3, "task.TaskName B: %s", task.TaskName)
	//if strings.HasSuffix(task.TaskName, "ERROR") {
	//	err = fmt.Errorf("(InitRaw) taskname is error")
	//	return
	//}

	if task.TaskName == "" && strings.HasPrefix(task.ID, job_prefix) {
		var tid Task_Unique_Identifier
		tid, err = New_Task_Unique_Identifier_FromString(task.ID)
		if err != nil {
			err = fmt.Errorf("(InitRaw) New_Task_Unique_Identifier_FromString returned: %s", err.Error())
			return
		}
		task.Task_Unique_Identifier = tid
		err = fmt.Errorf("(InitRaw) B) broken task? %s", jobID)
		return
	}

	var task_str string
	task_str, err = task.String()
	if err != nil {
		err = fmt.Errorf("(InitRaw) task.String returned: %s", err.Error())
		return
	}
	task.RWMutex.Init("task_" + task_str)

	// jobID is missing and task_id is only a number (e.g. on submission of old-style AWE)

	if task.TaskName == "" {
		err = fmt.Errorf("(InitRaw) task.TaskName empty")
		return
	}

	if task.ID != task_str {
		task.ID = task_str
		changed = true
	}

	if task.State == "" {
		task.State = TASK_STAT_INIT
		changed = true
	}

	if job != nil {
		if job.Info == nil {
			err = fmt.Errorf("(InitRaw) job.Info empty")
			return
		}
		task.Info = job.Info
	}

	if task.State != TASK_STAT_COMPLETED {
		if task.TotalWork == 0 {
			task.TotalWork = 1
			changed = true
		}
		if task.RemainWork != task.TotalWork {

			task.RemainWork = task.TotalWork
			changed = true
		}
	}

	if len(task.Cmd.Environ.Private) > 0 {
		task.Cmd.HasPrivateEnv = true
	}

	//if strings.HasPrefix(task.ID, task.JobId+"_") {
	//	task.ID = strings.TrimPrefix(task.ID, task.JobId+"_")
	//	changed = true
	//}

	//if strings.HasPrefix(task.ID, "_") {
	//	task.ID = strings.TrimPrefix(task.ID, "_")
	//	changed = true
	//}
	if job == nil {
		err = fmt.Errorf("(InitRaw) job is nil")
		return
	}
	var context *cwl.WorkflowContext

	if task.StepOutputInterface != nil {
		context = job.WorkflowContext
		task.StepOutput, err = cwl.NewJob_documentFromNamedTypes(task.StepOutputInterface, context)
		if err != nil {
			err = fmt.Errorf("(InitRaw) cwl.NewJob_documentFromNamedTypes returned: %s", err.Error())
			return
		}
	}

	// if CwlVersion != "" {
	// 	CwlVersion := context.CwlVersion
	// 	if task.CwlVersion != CwlVersion {
	// 		task.CwlVersion = CwlVersion
	// 	}
	// }

	if task.WorkflowStep != nil {

		if job != nil {
			if job.WorkflowContext == nil {
				err = fmt.Errorf("(InitRaw) job.WorkflowContext == nil")
				return
			}

			err = task.WorkflowStep.Init(job.WorkflowContext)
			if err != nil {
				err = fmt.Errorf("(InitRaw) task.WorkflowStep.Init returned: %s", err.Error())
				return
			}
		}
	}

	return
}

// Finalize this function prevents a dead-lock when a sub-workflow task finalizes
func (task *TaskRaw) Finalize() (ok bool, err error) {
	err = task.LockNamed("Finalize")
	if err != nil {
		return
	}
	defer task.Unlock()

	if task.Finalizing {
		// somebody else already flipped the bit
		ok = false
		return
	}

	task.Finalizing = true
	ok = true

	return

}

// IsValidUUID _
func IsValidUUID(uuid string) bool {
	if len(uuid) != 36 {
		return false
	}
	r := regexp.MustCompile("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$")
	return r.MatchString(uuid)
}

// IsProcessInstance _
func (task *Task) IsProcessInstance() {}

// CollectDependencies populate DependsOn
func (task *Task) CollectDependencies() (changed bool, err error) {

	deps := make(map[Task_Unique_Identifier]bool)
	depsChanged := false

	jobid, err := task.GetJobID()
	if err != nil {
		return
	}

	if jobid == "" {
		err = fmt.Errorf("(CollectDependencies) jobid is empty")
		return
	}

	jobPrefix := jobid + "_"

	// collect explicit dependencies
	for _, deptask := range task.DependsOn {

		if deptask == "" {
			depsChanged = true
			continue
		}

		if !strings.HasPrefix(deptask, jobPrefix) {
			deptask = jobPrefix + deptask
			depsChanged = true
		} else {
			deptaskSuffix := strings.TrimPrefix(deptask, jobPrefix)
			if deptaskSuffix == "" {
				depsChanged = true
				continue
			}
		}

		t, yerr := New_Task_Unique_Identifier_FromString(deptask)
		if yerr != nil {
			err = fmt.Errorf("(CollectDependencies) Cannot parse entry in DependsOn: %s", yerr.Error())
			return
		}

		if t.TaskName == "" {
			// this is to fix a bug
			depsChanged = true
			continue
		}

		deps[t] = true
	}

	for _, input := range task.Inputs {

		deptask := input.Origin
		if deptask == "" {
			depsChanged = true
			continue
		}

		if !strings.HasPrefix(deptask, jobPrefix) {
			deptask = jobPrefix + deptask
			depsChanged = true
		}

		t, yerr := New_Task_Unique_Identifier_FromString(deptask)
		if yerr != nil {

			err = fmt.Errorf("(CollectDependencies) Cannot parse Origin entry in Input: %s", yerr.Error())
			return

		}

		_, ok := deps[t]
		if !ok {
			// this was not yet in deps
			deps[t] = true
			depsChanged = true
		}

	}

	// write all dependencies if different from before
	if depsChanged {
		task.DependsOn = []string{}
		for deptask, _ := range deps {
			var dep_task_str string
			dep_task_str, err = deptask.String()
			if err != nil {
				err = fmt.Errorf("(CollectDependencies) dep_task.String returned: %s", err.Error())
				return
			}
			task.DependsOn = append(task.DependsOn, dep_task_str)
		}
		changed = true
	}
	return
}

// Init argument job is optional, but recommended
func (task *Task) Init(job *Job, job_id string) (changed bool, err error) {
	changed, err = task.InitRaw(job, job_id)
	if err != nil {
		return
	}

	depChanges, err := task.CollectDependencies()
	if err != nil {
		return
	}
	if depChanges {
		changed = true
	}

	// set node / host / url for files
	for _, io := range task.Inputs {
		if io.Node == "" {
			io.Node = "-"
		}
		_, err = io.DataUrl()
		if err != nil {
			return
		}
		logger.Debug(2, "inittask input: host="+io.Host+", node="+io.Node+", url="+io.Url)
	}
	for _, io := range task.Outputs {
		if io.Node == "" {
			io.Node = "-"
		}
		_, err = io.DataUrl()
		if err != nil {
			return
		}
		logger.Debug(2, "inittask output: host="+io.Host+", node="+io.Node+", url="+io.Url)
	}
	for _, io := range task.Predata {
		if io.Node == "" {
			io.Node = "-"
		}
		_, err = io.DataUrl()
		if err != nil {
			return
		}
		// predata IO can not be empty
		if (io.Url == "") && (io.Node == "-") {
			err = errors.New("Invalid IO, required fields url or host / node missing")
			return
		}
		logger.Debug(2, "inittask predata: host="+io.Host+", node="+io.Node+", url="+io.Url)
	}

	err = task.setTokenForIO(false)
	if err != nil {
		return
	}
	return
}

// NewTask task_id_str is without prefix yet
func NewTask(job *Job, workflowInstanceUUID string, workflowInstanceID string, task_id_str string) (t *Task, err error) {

	//fmt.Printf("(NewTask) new task: %s %s/%s\n", job.ID, workflow_instance_id, task_id_str)

	if task_id_str == "" {
		err = fmt.Errorf("(NewTask) task_id is empty")
		return

	}

	if strings.HasPrefix(task_id_str, job.Entrypoint) {
		err = fmt.Errorf("(NewTask) task_id_str prefix wrong: %s", task_id_str)
		return
	}

	if job.IsCWL {
		if task_id_str != job.Entrypoint {
			if !strings.HasPrefix(workflowInstanceID, job.Entrypoint) {
				err = fmt.Errorf("(NewTask) workflowInstanceID %s has not this prefix %s", workflowInstanceID, job.Entrypoint)
				return
			}
		}
	}

	if job.ID == "" {
		err = fmt.Errorf("(NewTask) jobid is empty! ")
		return
	}

	if strings.HasSuffix(task_id_str, "/") {
		err = fmt.Errorf("(NewTask) Suffix in task_id not ok %s", task_id_str)
		return
	}

	task_id_str = strings.TrimSuffix(task_id_str, "/")
	//workflow = strings.TrimSuffix(workflow, "/")

	var job_global_task_id_str string
	if job.IsCWL {
		job_global_task_id_str = workflowInstanceID + "/" + task_id_str
	} else {
		job_global_task_id_str = task_id_str
	}

	var tui Task_Unique_Identifier
	tui, err = New_Task_Unique_Identifier(job.ID, job_global_task_id_str)
	if err != nil {
		err = fmt.Errorf("(NewTask) New_Task_Unique_Identifier returns: %s", err.Error())
		return
	}

	var tr *TaskRaw
	tr, err = NewTaskRaw(tui, job.Info)
	if err != nil {
		err = fmt.Errorf("(NewTask) NewTaskRaw returns: %s", err.Error())
		return
	}
	t = &Task{
		TaskRaw: *tr,
		Inputs:  []*IO{},
		Outputs: []*IO{},
		Predata: []*IO{},
	}

	t.TaskRaw.WorkflowInstanceID = workflowInstanceID
	t.TaskRaw.WorkflowInstanceUUID = workflowInstanceUUID
	//if workflow_instance_id == "" {
	//	err = fmt.Errorf("(NewTask) workflow_instance_id empty")
	//	return

	//}

	return
}

// GetOutputs _
func (task *Task) GetOutputs() (outputs []*IO, err error) {
	outputs = []*IO{}

	lock, err := task.RLockNamed("GetOutputs")
	if err != nil {
		return
	}
	defer task.RUnlockNamed(lock)

	for _, output := range task.Outputs {
		outputs = append(outputs, output)
	}

	return
}

// GetOutput _
func (task *Task) GetOutput(filename string) (output *IO, err error) {
	lock, err := task.RLockNamed("GetOutput")
	if err != nil {
		return
	}
	defer task.RUnlockNamed(lock)

	for _, io := range task.Outputs {
		if io.FileName == filename {
			output = io
			return
		}
	}

	err = fmt.Errorf("Output %s not found", filename)
	return
}

// SetScatterChildren _
func (task *TaskRaw) SetScatterChildren(qm *ServerMgr, scatterChildren []string, writelock bool) (err error) {

	if writelock {
		err = task.LockNamed("SetScatterChildren")
		if err != nil {
			return
		}
		defer task.Unlock()
	}

	workflowInstanceID := task.WorkflowInstanceUUID
	if task.WorkflowInstanceID == "" {
		err = dbUpdateJobTaskField(task.JobId, task.WorkflowInstanceID, task.ID, "scatterChildren", scatterChildren)
		if err != nil {
			err = fmt.Errorf("(SetScatterChildren) dbUpdateJobTaskField returned: %s", err.Error())
			return
		}
	} else {
		err = dbUpdateTaskField(workflowInstanceID, task.ID, "scatterChildren", scatterChildren)
		if err != nil {
			err = fmt.Errorf("(SetScatterChildren) dbUpdateTaskField returned: %s", err.Error())
			return
		}
	}

	task.ScatterChildren = scatterChildren
	return
}

// GetScatterChildren _
func (task *TaskRaw) GetScatterChildren(wi *WorkflowInstance, qm *ServerMgr) (children []*Task, err error) {
	lock, err := task.RLockNamed("GetScatterChildren")
	if err != nil {
		return
	}
	defer task.RUnlockNamed(lock)

	if task.ScatterChildren_ptr != nil {
		children = task.ScatterChildren_ptr // should make a copy....
		return
	}

	children = []*Task{}
	for _, taskIDStr := range task.ScatterChildren {
		var child *Task
		var ok bool
		child, ok, err = wi.GetTaskByName(taskIDStr, true)
		if err != nil {
			err = fmt.Errorf("(GetScatterChildren) wi.GetTaskByName returned: %s", err.Error())
			return
		}
		if !ok {
			err = fmt.Errorf("(GetScatterChildren) child task %s not found in TaskMap", taskIDStr)
			return
		}
		children = append(children, child)
	}
	task.ScatterChildren_ptr = children

	return
}

// GetWorkflowInstance _
func (task *TaskRaw) GetWorkflowInstance(readLock bool) (wi *WorkflowInstance, ok bool, err error) {

	var job *Job
	job, err = task.GetJob(time.Second*90, readLock)
	if err != nil {
		err = fmt.Errorf("(GetWorkflowInstance) task.GetJob returned: %s", err.Error())
		return
	}

	wiID := task.WorkflowInstanceID
	wi, ok, err = job.GetWorkflowInstance(wiID, true)
	if err != nil {
		err = fmt.Errorf("(GetWorkflowInstance) job.GetWorkflowInstance returned: %s", err.Error())
		return
	}

	if !ok {
		err = fmt.Errorf("(GetWorkflowInstance) job.GetWorkflowInstance did not find: %s", wiID)
		return
	}

	return
}

// returns name of Parent (without jobid)
// func (task *TaskRaw) GetWorkflowParent() (p Task_Unique_Identifier, ok bool, err error) {
// 	lock, err := task.RLockNamed("GetParent")
// 	if err != nil {
// 		return
// 	}
// 	defer task.RUnlockNamed(lock)

// 	if task.WorkflowParent == nil {
// 		ok = false
// 		return
// 	}

// 	p = *task.WorkflowParent
// 	return
// }

// func (task *TaskRaw) GetWorkflowParentStr() (parent_id_str string, err error) {
// 	lock, err := task.RLockNamed("GetWorkflowParentStr")
// 	if err != nil {
// 		return
// 	}
// 	defer task.RUnlockNamed(lock)

// 	parent_id_str = ""

// 	if task.WorkflowParent != nil {
// 		parent_id_str, _ = task.WorkflowParent.String()
// 	}

// 	return
// }

// GetState _
func (task *TaskRaw) GetState() (state string, err error) {
	lock, err := task.RLockNamed("GetState")
	if err != nil {
		return
	}
	defer task.RUnlockNamed(lock)
	state = task.State
	return
}

// GetStateTimeout _
func (task *TaskRaw) GetStateTimeout(timeout time.Duration) (state string, err error) {
	lock, err := task.RLockNamedTimeout("GetStateTimeout", timeout)
	if err != nil {
		return
	}
	defer task.RUnlockNamed(lock)
	state = task.State
	return
}

// // GetTaskType _
// func (task *TaskRaw) GetTaskType() (type_str string, err error) {
// 	lock, err := task.RLockNamed("GetTaskType")
// 	if err != nil {
// 		return
// 	}
// 	defer task.RUnlockNamed(lock)
// 	type_str = task.TaskType
// 	return
// }

// SetProcessType _ _
func (task *TaskRaw) SetProcessType(typeStr string, doSync bool, writelock bool) (err error) {
	if writelock {
		err = task.LockNamed("SetProcessType")
		if err != nil {
			return
		}
		defer task.Unlock()
	}
	if doSync {
		if task.WorkflowInstanceID == "" {
			err = dbUpdateJobTaskString(task.JobId, task.WorkflowInstanceID, task.ID, "processtype", typeStr)
			if err != nil {
				err = fmt.Errorf("(task/SetProcessType) dbUpdateJobTaskString returned: %s", err.Error())
				return
			}
		} else {
			err = dbUpdateTaskString(task.WorkflowInstanceUUID, task.ID, "processtype", typeStr)
			if err != nil {
				err = fmt.Errorf("(task/SetProcessType) dbUpdateTaskString returned: %s", err.Error())
				return
			}
		}
	}
	task.ProcessType = typeStr
	return
}

// SetTaskNotReadyReason _
func (task *Task) SetTaskNotReadyReason(reason string, writelock bool) (err error) {
	if writelock {
		err = task.LockNamed("SetTaskNotReadyReason")
		if err != nil {
			return
		}
		defer task.Unlock()
	}

	if task.NotReadyReason == reason {
		return
	}

	if task.WorkflowInstanceID == "" {
		err = dbUpdateJobTaskString(task.JobId, task.WorkflowInstanceID, task.ID, "notReadyReason", reason)
		if err != nil {
			err = fmt.Errorf("(task/SetTaskNotReadyReason) dbUpdateJobTaskString returned: %s", err.Error())
			return
		}
	} else {
		err = dbUpdateTaskString(task.WorkflowInstanceUUID, task.ID, "notReadyReason", reason)
		if err != nil {
			err = fmt.Errorf("(task/SetTaskNotReadyReason) dbUpdateTaskString returned: %s", err.Error())
			return
		}
	}

	task.NotReadyReason = reason
	return
}

func (task *TaskRaw) SetCreatedDate(t time.Time) (err error) {
	err = task.LockNamed("SetCreatedDate")
	if err != nil {
		return
	}
	defer task.Unlock()

	if task.WorkflowInstanceID == "" {
		err = dbUpdateJobTaskTime(task.JobId, task.WorkflowInstanceID, task.ID, "createdDate", t)
		if err != nil {
			err = fmt.Errorf("(task/SetCreatedDate) dbUpdateJobTaskTime returned: %s", err.Error())
			return
		}
	} else {
		err = dbUpdateTaskTime(task.WorkflowInstanceUUID, task.ID, "createdDate", t)
		if err != nil {
			err = fmt.Errorf("(task/SetCreatedDate) dbUpdateTaskTime returned: %s", err.Error())
			return
		}
	}

	task.CreatedDate = t
	return
}

func (task *TaskRaw) SetStartedDate(t time.Time) (err error) {
	err = task.LockNamed("SetStartedDate")
	if err != nil {
		return
	}
	defer task.Unlock()

	if task.WorkflowInstanceID == "" {
		err = dbUpdateJobTaskTime(task.JobId, task.WorkflowInstanceID, task.ID, "startedDate", t)
		if err != nil {
			err = fmt.Errorf("(task/SetStartedDate) dbUpdateJobTaskTime returned: %s", err.Error())
			return
		}
	} else {
		err = dbUpdateTaskTime(task.WorkflowInstanceUUID, task.ID, "startedDate", t)
		if err != nil {
			err = fmt.Errorf("(task/SetStartedDate) dbUpdateTaskTime returned: %s", err.Error())
			return
		}
	}
	task.StartedDate = t
	return
}

func (task *TaskRaw) SetCompletedDate(t time.Time, lock bool) (err error) {
	if lock {
		err = task.LockNamed("SetCompletedDate")
		if err != nil {
			return
		}
		defer task.Unlock()
	}
	if task.WorkflowInstanceID == "" {
		err = dbUpdateJobTaskTime(task.JobId, task.WorkflowInstanceID, task.ID, "completedDate", t)
		if err != nil {
			err = fmt.Errorf("(task/SetCompletedDate) dbUpdateJobTaskTime returned: %s", err.Error())
			return
		}
	} else {
		err = dbUpdateTaskTime(task.WorkflowInstanceUUID, task.ID, "completedDate", t)
		if err != nil {
			err = fmt.Errorf("(task/SetCompletedDate) dbUpdateTaskTime returned: %s", err.Error())
			return
		}
	}
	task.CompletedDate = t
	return
}

// SetStepOutput _
func (task *TaskRaw) SetStepOutput(jd cwl.Job_document, lock bool) (err error) {
	if lock {
		err = task.LockNamed("SetStepOutput")
		if err != nil {
			return
		}
		defer task.Unlock()
	}
	if task.WorkflowInstanceID == "" {
		err = dbUpdateJobTaskField(task.JobId, task.WorkflowInstanceID, task.ID, "stepOutput", jd)
		if err != nil {
			err = fmt.Errorf("(task/SetStepOutput) dbUpdateJobTaskField returned: %s", err.Error())
			return
		}
	} else {
		err = dbUpdateTaskField(task.WorkflowInstanceUUID, task.ID, "stepOutput", jd)
		if err != nil {
			err = fmt.Errorf("(task/SetStepOutput) dbUpdateTaskField returned: %s", err.Error())
			return
		}
	}
	task.StepOutput = &jd
	task.StepOutputInterface = jd
	return
}

// // SetProcessType _ duplicate
// func (task *TaskRaw) SetProcessType(t string, doSync bool, lock bool) (err error) {
// 	if lock {
// 		err = task.LockNamed("SetProcessType")
// 		if err != nil {
// 			return
// 		}
// 		defer task.Unlock()
// 	}

// 	if doSync {
// 		err = dbUpdateTaskString(task.WorkflowInstanceUUID, task.ID, "processtype", t)
// 		if err != nil {
// 			err = fmt.Errorf("(task/SetProcessOutput) dbUpdateTaskField returned: %s", err.Error())
// 			return
// 		}
// 	}
// 	task.ProcessType = t
// 	return
// }

// SetProcessOutput Process is CommandLineTool, ExpressionTool or Workflow
func (task *TaskRaw) SetProcessOutput(jd cwl.Job_document, lock bool) (err error) {
	if lock {
		err = task.LockNamed("SetProcessOutput")
		if err != nil {
			return
		}
		defer task.Unlock()
	}
	if task.WorkflowInstanceID == "" {
		err = dbUpdateJobTaskField(task.JobId, task.WorkflowInstanceID, task.ID, "processOutput", &jd)
		if err != nil {
			err = fmt.Errorf("(task/SetProcessOutput) dbUpdateJobTaskField returned: %s", err.Error())
			return
		}
	} else {
		err = dbUpdateTaskField(task.WorkflowInstanceUUID, task.ID, "processOutput", &jd)
		if err != nil {
			err = fmt.Errorf("(task/SetProcessOutput) dbUpdateTaskField returned: %s", err.Error())
			return
		}
	}
	task.ProcessOutput = &jd
	task.ProcessOutputInterface = jd
	return
}

// only for debugging purposes
// GetStateNamed only for debugging purposes
func (task *TaskRaw) GetStateNamed(name string) (state string, err error) {
	lock, err := task.RLockNamed("GetState/" + name)
	if err != nil {
		return
	}
	defer task.RUnlockNamed(lock)
	state = task.State
	return
}

// GetStateNamedTimeout _
func (task *TaskRaw) GetStateNamedTimeout(name string, timeout time.Duration) (state string, err error) {
	lock, err := task.RLockNamedTimeout("GetState/"+name, timeout)
	if err != nil {
		return
	}
	defer task.RUnlockNamed(lock)
	state = task.State
	return
}

// GetID _
func (task *TaskRaw) GetID(me string) (id Task_Unique_Identifier, err error) {
	lock, err := task.RLockNamed("GetId:" + me)
	if err != nil {
		return
	}
	defer task.RUnlockNamed(lock)
	id = task.Task_Unique_Identifier
	return
}

// GetIDTimeout _
func (task *TaskRaw) GetIDTimeout(me string, timeout time.Duration) (id Task_Unique_Identifier, err error) {
	lock, err := task.RLockNamedTimeout("GetIdTimeout:"+me, timeout)
	if err != nil {
		return
	}
	defer task.RUnlockNamed(lock)
	id = task.Task_Unique_Identifier
	return
}

// GetJobID _
func (task *TaskRaw) GetJobID() (id string, err error) {
	lock, err := task.RLockNamed("GetJobId")
	if err != nil {
		return
	}
	defer task.RUnlockNamed(lock)
	id = task.JobId
	return
}

// GetJob _
func (task *TaskRaw) GetJob(timeout time.Duration, readLock bool) (job *Job, err error) {
	if readLock {
		var lock rwmutex.ReadLock
		lock, err = task.RLockNamedTimeout("GetJob", timeout)
		if err != nil {
			return
		}
		defer task.RUnlockNamed(lock)
	}
	if task.job != nil {
		job = task.job
		return
	}

	jobid := task.JobId

	job, err = GetJob(jobid)
	if err != nil {
		err = fmt.Errorf("(TaskRaw/GetJob) global GetJob returned: %s", err.Error())
		return
	}

	// this is writing while we just have a readlock, not so nice
	task.job = job

	return
}

// SetState also updates wi.RemainTasks, task.SetCompletedDate
func (task *TaskRaw) SetState(newState string, writeLock bool, caller string) (err error) {
	if writeLock {
		err = task.LockNamed("TaskRaw/SetState/" + caller)
		if err != nil {
			return
		}
		defer task.Unlock()
	}

	oldState := task.State
	taskid := task.ID
	jobid := task.JobId

	if jobid == "" {
		err = fmt.Errorf("task %s has no job id", taskid)
		return
	}
	if oldState == newState {
		return
	}

	if task.WorkflowInstanceID == "" {

		err = dbUpdateJobTaskString(jobid, task.WorkflowInstanceID, taskid, "state", newState)
		if err != nil {
			err = fmt.Errorf("(TaskRaw/SetState) dbUpdateJobTaskString returned: %s", err.Error())
			return
		}
	} else {

		if task.WorkflowInstanceUUID == "" {
			err = fmt.Errorf("(TaskRaw/SetState) task.WorkflowInstanceUUID is empty")
			return
		}

		err = dbUpdateTaskString(task.WorkflowInstanceUUID, taskid, "state", newState)
		if err != nil {
			err = fmt.Errorf("(TaskRaw/SetState) dbUpdateTaskString returned: %s", err.Error())
			return
		}
	}

	logger.Debug(3, "(TaskRaw/SetState) %s new state: \"%s\" (old state \"%s\")", taskid, newState, oldState)
	task.State = newState

	if newState == TASK_STAT_COMPLETED {
		var wi *WorkflowInstance
		var ok bool
		wi, ok, err = task.GetWorkflowInstance(false)
		if err != nil {
			err = fmt.Errorf("(TaskRaw/SetState) GetWorkflowInstance returned: %s", err.Error())
			return
		}
		if !ok {
			err = fmt.Errorf("(TaskRaw/SetState) WorkflowInstance not found")
			return
		}

		if wi != nil {
			_, err = wi.IncrementRemainSteps(-1, true)
			if err != nil {
				err = fmt.Errorf("(TaskRaw/SetState) wi.DecreaseRemainSteps returned: %s", err.Error())
				return
			}
		}

		err = task.SetCompletedDate(time.Now(), false)
		if err != nil {
			err = fmt.Errorf("(TaskRaw/SetState) task.SetCompletedDate returned: %s", err.Error())
			return
		}

	} else if oldState == TASK_STAT_COMPLETED {
		// in case a completed task is marked as something different
		//var job *Job
		//job, err = GetJob(jobid)
		//if err != nil {
		//	return
		//}

		//_, err = job.IncrementRemainSteps(1, "TaskRaw/SetState")
		//if err != nil {
		//	err = fmt.Errorf("(TaskRaw/SetState) IncrementRemainSteps returned: %s", err.Error())
		//	return
		//}
		initTime := time.Time{}
		err = task.SetCompletedDate(initTime, false)
		if err != nil {
			err = fmt.Errorf("(TaskRaw/SetState) SetCompletedDate returned: %s", err.Error())
			return
		}

	}

	return
}

// GetDependsOn _
func (task *TaskRaw) GetDependsOn() (dep []string, err error) {
	lock, err := task.RLockNamed("GetDependsOn")
	if err != nil {
		return
	}
	defer task.RUnlockNamed(lock)
	dep = task.DependsOn
	return
}

// CreateInputIndexes checks and creates indices on input shock nodes if needed
func (task *Task) CreateInputIndexes() (err error) {
	for _, io := range task.Inputs {
		_, err = io.IndexFile(io.ShockIndex)
		if err != nil {
			err = fmt.Errorf("(CreateInputIndexes) failed to create shock index: node=%s, taskid=%s, error=%s", io.Node, task.ID, err.Error())
			logger.Error(err.Error())
			return
		}
	}
	return
}

// CreateOutputIndexes _
// checks and creates indices on output shock nodes if needed
// if worker failed to do so, this will catch it
func (task *Task) CreateOutputIndexes() (err error) {
	for _, io := range task.Outputs {
		_, err = io.IndexFile(io.ShockIndex)
		if err != nil {
			err = fmt.Errorf("(CreateOutputIndexes) failed to create shock index: node=%s, taskid=%s, error=%s", io.Node, task.ID, err.Error())
			logger.Error(err.Error())
			return
		}
	}
	return
}

// check that part index is valid before initalizing it
// refactored out of InitPartIndex deal with potentailly long write lock
func (task *Task) checkPartIndex() (newPartition *PartInfo, totalunits int, isSingle bool, err error) {
	lock, err := task.RLockNamed("checkPartIndex")
	if err != nil {
		return
	}
	defer task.RUnlockNamed(lock)

	if task.Inputs == nil {
		err = fmt.Errorf("(checkPartIndex) no inputs found")
		return
	}

	if len(task.Inputs) == 0 {
		err = fmt.Errorf("(checkPartIndex) array task.Inputs empty")
		return
	}

	inputIO := task.Inputs[0]
	newPartition = &PartInfo{
		Input:         inputIO.FileName,
		MaxPartSizeMB: task.MaxWorkSize,
	}

	if len(task.Inputs) > 1 {
		found := false
		if (task.Partition != nil) && (task.Partition.Input != "") {
			// task submitted with partition input specified, use that
			for _, io := range task.Inputs {
				if io.FileName == task.Partition.Input {
					found = true
					inputIO = io
					newPartition.Input = io.FileName
				}
			}
		}
		if !found {
			// bad state - set as not multi-workunit
			logger.Error("warning: lacking partition info while multiple inputs are specified, taskid=" + task.ID)
			isSingle = true
			return
		}
	}

	// if submitted with partition index use that, otherwise default
	if (task.Partition != nil) && (task.Partition.Index != "") {
		newPartition.Index = task.Partition.Index
	} else {
		newPartition.Index = conf.DEFAULT_INDEX
	}

	idxInfo, err := inputIO.IndexFile(newPartition.Index)
	if err != nil {
		// bad state - set as not multi-workunit
		logger.Error("warning: failed to create / retrieve index=%s, taskid=%s, error=%s", newPartition.Index, task.ID, err.Error())
		isSingle = true
		err = nil
		return
	}

	totalunits = int(idxInfo.TotalUnits)
	return
}

// InitPartIndex _
// get part size based on partition/index info
// this resets task.Partition when called
// only 1 task.Inputs allowed unless 'partinfo.input' specified on POST
// if fail to get index info, task.TotalWork set to 1 and task.Partition set to nil
func (task *Task) InitPartIndex() (err error) {
	if task.TotalWork == 1 && task.MaxWorkSize == 0 {
		// only 1 workunit requested
		return
	}

	newPartition, totalunits, isSingle, err := task.checkPartIndex()
	if err != nil {
		return
	}
	if isSingle {
		// its a single workunit, skip init
		err = task.setSingleWorkunit(true)
		return
	}

	// err = task.LockNamed("InitPartIndex")
	// if err != nil {
	// 	return
	// }
	// defer task.Unlock()

	// adjust total work based on needs
	if newPartition.MaxPartSizeMB > 0 {
		// this implementation for chunkrecord indexer only
		chunkmb := int(conf.DEFAULT_CHUNK_SIZE / 1048576)
		var totalwork int
		if totalunits*chunkmb%newPartition.MaxPartSizeMB == 0 {
			totalwork = totalunits * chunkmb / newPartition.MaxPartSizeMB
		} else {
			totalwork = totalunits*chunkmb/newPartition.MaxPartSizeMB + 1
		}
		if totalwork < task.TotalWork {
			// use bigger splits (specified by size or totalwork)
			totalwork = task.TotalWork
		}

		if totalwork == 0 {
			totalwork = 1 // need at least one unit for empty file
		}

		if totalwork != task.TotalWork {
			err = task.setTotalWork(totalwork, true)
			if err != nil {
				return
			}
		}
	}

	if totalunits == 0 {
		totalunits = 1
	}

	if totalunits < task.TotalWork {
		err = task.setTotalWork(totalunits, true)
		if err != nil {
			return
		}
	}

	// need only 1 workunit
	if task.TotalWork == 1 {
		err = task.setSingleWorkunit(true)
		return
	}

	// done, set it
	newPartition.TotalIndex = totalunits
	err = task.setPartition(newPartition, false)
	return
}

// wrapper functions to set: totalwork=1, partition=nil, maxworksize=0
func (task *Task) setSingleWorkunit(writelock bool) (err error) {
	if task.TotalWork != 1 {
		err = task.setTotalWork(1, writelock)
		if err != nil {
			err = fmt.Errorf("(task/setSingleWorkunit) setTotalWork returned: %s", err.Error())
			return
		}
	}
	if task.Partition != nil {
		err = task.setPartition(nil, writelock)
		if err != nil {
			err = fmt.Errorf("(task/setSingleWorkunit) setPartition returned: %s", err.Error())
			return
		}
	}
	if task.MaxWorkSize != 0 {
		err = task.setMaxWorkSize(0, writelock)
		if err != nil {
			err = fmt.Errorf("(task/setSingleWorkunit) setMaxWorkSize returned: %s", err.Error())
			return
		}
	}
	return
}

func (task *Task) setTotalWork(num int, writelock bool) (err error) {
	if writelock {
		err = task.LockNamed("setTotalWork")
		if err != nil {
			return
		}
		defer task.Unlock()
	}
	err = dbUpdateJobTaskInt(task.JobId, task.WorkflowInstanceID, task.ID, "totalwork", num)
	if err != nil {
		err = fmt.Errorf("(task/setTotalWork) dbUpdateJobTaskInt returned: %s", err.Error())
		return
	}
	task.TotalWork = num
	// reset remaining work whenever total work reset
	err = task.SetRemainWork(num, false)
	return
}

func (task *Task) setPartition(partition *PartInfo, writelock bool) (err error) {
	if writelock {
		err = task.LockNamed("setPartition")
		if err != nil {
			return
		}
		defer task.Unlock()
	}
	err = dbUpdateJobTaskPartition(task.JobId, task.WorkflowInstanceID, task.ID, partition)
	if err != nil {
		err = fmt.Errorf("(task/setPartition) dbUpdateJobTaskPartition returned: %s", err.Error())
		return
	}
	task.Partition = partition
	return
}

func (task *Task) setMaxWorkSize(num int, writelock bool) (err error) {
	if writelock {
		err = task.LockNamed("setMaxWorkSize")
		if err != nil {
			return
		}
		defer task.Unlock()
	}
	err = dbUpdateJobTaskInt(task.JobId, task.WorkflowInstanceID, task.ID, "maxworksize", num)
	if err != nil {
		err = fmt.Errorf("(task/setMaxWorkSize) dbUpdateJobTaskInt returned: %s", err.Error())
		return
	}
	task.MaxWorkSize = num
	return
}

// SetRemainWork _
func (task *Task) SetRemainWork(num int, writelock bool) (err error) {
	if writelock {
		err = task.LockNamed("SetRemainWork")
		if err != nil {
			return
		}
		defer task.Unlock()
	}
	err = dbUpdateJobTaskInt(task.JobId, task.WorkflowInstanceID, task.ID, "remainwork", num)
	if err != nil {
		err = fmt.Errorf("(task/SetRemainWork) dbUpdateJobTaskInt returned: %s", err.Error())
		return
	}
	task.RemainWork = num
	return
}

// IncrementRemainWork _
func (task *Task) IncrementRemainWork(inc int, writelock bool) (remainwork int, err error) {
	if writelock {
		err = task.LockNamed("IncrementRemainWork")
		if err != nil {
			return
		}
		defer task.Unlock()
	}

	remainwork = task.RemainWork + inc

	if remainwork < 0 {
		err = fmt.Errorf("(task/IncrementRemainWork) remainwork < 0")
		return
	}

	if task.WorkflowInstanceID == "" {
		err = dbUpdateJobTaskInt(task.JobId, task.WorkflowInstanceID, task.ID, "remainwork", remainwork)
		if err != nil {
			err = fmt.Errorf("(task/IncrementRemainWork) dbUpdateJobTaskInt returned: %s", err.Error())
			return
		}
	} else {
		err = dbUpdateTaskInt(task.WorkflowInstanceUUID, task.ID, "remainwork", remainwork)
		if err != nil {
			err = fmt.Errorf("(task/IncrementRemainWork) dbUpdateTaskInt returned: %s", err.Error())
			return
		}
	}
	task.RemainWork = remainwork
	return
}

// IncrementComputeTime _
func (task *Task) IncrementComputeTime(inc int) (err error) {
	err = task.LockNamed("IncrementComputeTime")
	if err != nil {
		return
	}
	defer task.Unlock()

	newComputeTime := task.ComputeTime + inc

	if task.WorkflowInstanceID == "" {
		err = dbUpdateJobTaskInt(task.JobId, task.WorkflowInstanceID, task.ID, "computetime", newComputeTime)
		if err != nil {
			err = fmt.Errorf("(task/IncrementComputeTime) dbUpdateJobTaskInt returned: %s", err.Error())
			return
		}
	} else {
		err = dbUpdateTaskInt(task.WorkflowInstanceUUID, task.ID, "computetime", newComputeTime)
		if err != nil {
			err = fmt.Errorf("(task/IncrementComputeTime) dbUpdateTaskInt returned: %s", err.Error())
			return
		}

	}
	task.ComputeTime = newComputeTime
	return
}

// ResetTaskTrue _
func (task *Task) ResetTaskTrue(name string) (err error) {
	if task.ResetTask == true {
		return
	}
	err = task.LockNamed("ResetTaskTrue:" + name)
	if err != nil {
		return
	}
	defer task.Unlock()

	err = task.SetState(TASK_STAT_PENDING, false, "task/ResetTaskTrue")
	if err != nil {
		err = fmt.Errorf("(task/ResetTaskTrue) task.SetState returned: %s", err.Error())
		return
	}
	if task.WorkflowInstanceID == "" {
		err = dbUpdateJobTaskBoolean(task.JobId, task.WorkflowInstanceID, task.ID, "resettask", true)
		if err != nil {
			err = fmt.Errorf("(task/ResetTaskTrue) dbUpdateJobTaskBoolean returned: %s", err.Error())
			return
		}
	} else {
		err = dbUpdateTaskBoolean(task.WorkflowInstanceUUID, task.ID, "resettask", true)
		if err != nil {
			err = fmt.Errorf("(task/ResetTaskTrue) dbUpdateTaskBoolean returned: %s", err.Error())
			return
		}
	}
	task.ResetTask = true
	return
}

// SetResetTask _
func (task *Task) SetResetTask(info *Info) (err error) {
	// called when enqueing a task that previously ran

	// only run if true
	if task.ResetTask == false {
		return
	}

	// in memory pointer
	if task.Info == nil {
		task.Info = info
	}

	// reset remainwork
	err = task.SetRemainWork(task.TotalWork, true)
	if err != nil {
		err = fmt.Errorf("(task/SetResetTask) task.SetRemainWork returned: %s", err.Error())
		return
	}

	// reset computetime
	if task.WorkflowInstanceID == "" {
		err = dbUpdateJobTaskInt(task.JobId, task.WorkflowInstanceID, task.ID, "computetime", 0)
		if err != nil {
			err = fmt.Errorf("(task/SetResetTask) dbUpdateJobTaskInt returned: %s", err.Error())
			return
		}
	} else {
		err = dbUpdateTaskInt(task.WorkflowInstanceUUID, task.ID, "computetime", 0)
		if err != nil {
			err = fmt.Errorf("(task/SetResetTask) dbUpdateTaskInt returned: %s", err.Error())
			return
		}
	}
	task.ComputeTime = 0 // TODO use SetComputeTime()

	// reset completedate
	err = task.SetCompletedDate(time.Time{}, true)
	if err != nil {
		err = fmt.Errorf("(task/SetResetTask) SetCompletedDate returned: %s", err.Error())
		return
	}

	err = task.SetResetTaskInputs()
	if err != nil {
		err = fmt.Errorf("(task/SetResetTask) SetResetTaskInputs returned: %s", err.Error())
		return
	}

	err = task.SetResetTaskOutputs()
	if err != nil {
		err = fmt.Errorf("(task/SetResetTask) SetResetTaskOutputs returned: %s", err.Error())
		return
	}

	// delete all workunit logs
	for _, log := range conf.WORKUNIT_LOGS {
		err = task.DeleteLogs(log, true)
		if err != nil {
			return
		}
	}

	// reset the reset
	if task.WorkflowInstanceID == "" {
		err = dbUpdateJobTaskBoolean(task.JobId, task.WorkflowInstanceID, task.ID, "resettask", false)
		if err != nil {
			err = fmt.Errorf("(task/SetResetTask) dbUpdateJobTaskBoolean returned: %s", err.Error())
			return
		}
	} else {
		err = dbUpdateTaskBoolean(task.WorkflowInstanceUUID, task.ID, "resettask", false)
		if err != nil {
			err = fmt.Errorf("(task/SetResetTask) dbUpdateTaskBoolean returned: %s", err.Error())
			return
		}
	}

	err = task.LockNamed("SetResetTask")
	if err != nil {
		return
	}
	defer task.Unlock()

	task.ResetTask = false
	return
}

// SetResetTaskOutputs _
func (task *Task) SetResetTaskOutputs() (err error) {
	err = task.LockNamed("SetResetTaskOutputs")
	if err != nil {
		return
	}
	defer task.Unlock()

	// reset / delete all outputs
	for _, io := range task.Outputs {
		// do not delete update IO
		if io.Type == "update" {
			continue
		}
		if dataUrl, _ := io.DataUrl(); dataUrl != "" {
			// delete dataUrl if is shock node
			if strings.HasSuffix(dataUrl, shock.DATA_SUFFIX) {
				err = shock.ShockDelete(io.Host, io.Node, io.DataToken)
				if err == nil {
					logger.Debug(2, "Deleted node %s from shock", io.Node)
				} else {
					logger.Error("(SetResetTask) unable to deleted node %s from shock: %s", io.Node, err.Error())
				}
			}
		}
		io.Node = "-"
		io.Size = 0
		io.Url = ""
	}
	if task.WorkflowInstanceID == "" {
		err = dbUpdateJobTaskIO(task.JobId, task.WorkflowInstanceID, task.ID, "outputs", task.Outputs)
		if err != nil {
			err = fmt.Errorf("(task/SetResetTask) dbUpdateJobTaskIO returned: %s", err.Error())
			return
		}
	} else {
		err = dbUpdateTaskIO(task.WorkflowInstanceUUID, task.ID, "outputs", task.Outputs)
		if err != nil {
			err = fmt.Errorf("(task/SetResetTask) dbUpdateTaskIO returned: %s", err.Error())
			return
		}
	}

	return
}

// SetResetTaskInputs _
func (task *Task) SetResetTaskInputs() (err error) {
	err = task.LockNamed("SetResetTaskInputs")
	if err != nil {
		return
	}
	defer task.Unlock()

	// reset inputs
	for _, io := range task.Inputs {
		// skip inputs with no origin (predecessor task)
		if io.Origin == "" {
			continue
		}
		io.Node = "-"
		io.Size = 0
		io.Url = ""
	}
	if task.WorkflowInstanceID == "" {
		err = dbUpdateJobTaskIO(task.JobId, task.WorkflowInstanceID, task.ID, "inputs", task.Inputs)
		if err != nil {
			err = fmt.Errorf("(task/SetResetTask) dbUpdateJobTaskIO returned: %s", err.Error())
			return
		}
	} else {
		err = dbUpdateTaskIO(task.WorkflowInstanceUUID, task.ID, "inputs", task.Inputs)
		if err != nil {
			err = fmt.Errorf("(task/SetResetTask) dbUpdateTaskIO returned: %s", err.Error())
			return
		}
	}

	return
}

func (task *Task) setTokenForIO(writelock bool) (err error) {
	if writelock {
		err = task.LockNamed("setTokenForIO")
		if err != nil {
			return
		}
		defer task.Unlock()
	}
	if task.Info == nil {
		err = fmt.Errorf("(setTokenForIO) task.Info empty")
		return
	}
	if !task.Info.Auth || task.Info.DataToken == "" {
		return
	}
	// update inputs
	changed := false
	for _, io := range task.Inputs {
		if io.DataToken != task.Info.DataToken {
			io.DataToken = task.Info.DataToken
			changed = true
		}
	}
	if changed {
		if task.WorkflowInstanceID == "" {
			err = dbUpdateJobTaskIO(task.JobId, task.WorkflowInstanceID, task.ID, "inputs", task.Inputs)
			if err != nil {
				err = fmt.Errorf("(task/setTokenForIO) dbUpdateJobTaskIO returned: %s", err.Error())
				return
			}
		} else {
			err = dbUpdateTaskIO(task.WorkflowInstanceUUID, task.ID, "inputs", task.Inputs)
			if err != nil {
				err = fmt.Errorf("(task/setTokenForIO) dbUpdateTaskIO returned: %s", err.Error())
				return
			}
		}
	}
	// update outputs
	changed = false
	for _, io := range task.Outputs {
		if io.DataToken != task.Info.DataToken {
			io.DataToken = task.Info.DataToken
			changed = true
		}
	}
	if changed {
		if task.WorkflowInstanceID == "" {
			err = dbUpdateJobTaskIO(task.JobId, task.WorkflowInstanceID, task.ID, "outputs", task.Outputs)
			if err != nil {
				err = fmt.Errorf("(task/setTokenForIO) dbUpdateJobTaskIO returned: %s", err.Error())
				return
			}
		} else {
			err = dbUpdateTaskIO(task.WorkflowInstanceUUID, task.ID, "outputs", task.Outputs)
			if err != nil {
				err = fmt.Errorf("(task/setTokenForIO) dbUpdateTaskIO returned: %s", err.Error())
				return
			}
		}
	}
	return
}

// CreateWorkunits _
func (task *Task) CreateWorkunits(qm *ServerMgr, job *Job) (wus []*Workunit, err error) {
	//if a task contains only one workunit, assign rank 0

	//if task.WorkflowStep != nil {
	//	step :=
	//}
	//task_name := task.TaskName
	//fmt.Printf("(CreateWorkunits) %s\n", task_name)
	//if task_name == "_root__main_step1_step1_0" {
	//	panic("found task")
	//}

	if task.TotalWork == 0 {
		err = fmt.Errorf("(CreateWorkunits) task.TotalWork == 0")
		return
	}

	if task.TotalWork == 1 {
		var workunit *Workunit
		workunit, err = NewWorkunit(qm, task, 0, job)
		if err != nil {
			err = fmt.Errorf("(CreateWorkunits) (single) NewWorkunit failed: %s", err.Error())
			return
		}
		wus = append(wus, workunit)
		return
	}
	// if a task contains N (N>1) workunits, assign rank 1..N
	for i := 1; i <= task.TotalWork; i++ {
		var workunit *Workunit
		workunit, err = NewWorkunit(qm, task, i, job)
		if err != nil {
			err = fmt.Errorf("(CreateWorkunits) (multi) NewWorkunit failed: %s", err.Error())
			return
		}
		wus = append(wus, workunit)
	}
	return
}

// GetTaskLogs _
func (task *Task) GetTaskLogs() (tlog *TaskLog, err error) {
	tlog = new(TaskLog)
	tlog.ID = task.ID
	tlog.State = task.State
	tlog.TotalWork = task.TotalWork
	tlog.CompletedDate = task.CompletedDate

	workunitID := New_Workunit_Unique_Identifier(task.Task_Unique_Identifier, 0)
	//workunit_id := Workunit_Unique_Identifier{JobId: task.JobId, TaskName: task.ID}

	if task.TotalWork == 1 {
		//workunit_id.Rank = 0
		var wl *WorkLog
		wl, err = NewWorkLog(workunitID)
		if err != nil {
			err = fmt.Errorf("(task/GetTaskLogs) NewWorkLog returned: %s", err.Error())
			return
		}
		tlog.Workunits = append(tlog.Workunits, wl)
	} else {
		for i := 1; i <= task.TotalWork; i++ {
			workunitID.Rank = i
			var wl *WorkLog
			wl, err = NewWorkLog(workunitID)
			if err != nil {
				err = fmt.Errorf("(task/GetTaskLogs) NewWorkLog returned: %s", err.Error())
				return
			}
			tlog.Workunits = append(tlog.Workunits, wl)
		}
	}
	return
}

// ValidateDependants _
func (task *Task) ValidateDependants(qm *ServerMgr) (reason string, skip bool, err error) {
	lock, err := task.RLockNamed("ValidateDependants")
	if err != nil {
		return
	}
	defer task.RUnlockNamed(lock)

	// validate task states in depends on list
	for _, preTaskStr := range task.DependsOn {
		var preId Task_Unique_Identifier
		preId, err = New_Task_Unique_Identifier_FromString(preTaskStr)
		if err != nil {
			err = fmt.Errorf("(ValidateDependants) (field DependsOn) New_Task_Unique_Identifier_FromString returns: %s", err.Error())
			return
		}
		preTask, ok, xerr := qm.TaskMap.Get(preId, true)
		if xerr != nil {
			err = fmt.Errorf("(ValidateDependants) (field DependsOn) predecessor task %s not found for task %s: %s", preTaskStr, task.ID, xerr.Error())
			return
		}
		if !ok {
			reason = fmt.Sprintf("(ValidateDependants) (field DependsOn) predecessor task not found: task=%s, pretask=%s", task.ID, preTaskStr)
			logger.Debug(3, reason)
			return
		}
		var preTaskState string
		preTaskState, err = preTask.GetStateTimeout(time.Second * 1)
		if err != nil {
			err = nil
			skip = true
			//err = fmt.Errorf("(ValidateDependants) (field DependsOn) unable to get state for predecessor task %s: %s", preTaskStr, xerr.Error())
			return
		}
		if preTaskState != TASK_STAT_COMPLETED {
			reason = fmt.Sprintf("(ValidateDependants) (field DependsOn) predecessor task state is not completed: task=%s, pretask=%s, pretask.state=%s", task.ID, preTaskStr, preTaskState)
			logger.Debug(3, reason)
			return
		}
	}

	// validate task states in input IO origins
	for _, io := range task.Inputs {
		if io.Origin == "" {
			continue
		}
		var preId Task_Unique_Identifier
		preId, err = New_Task_Unique_Identifier(task.JobId, io.Origin)
		if err != nil {
			err = fmt.Errorf("(ValidateDependants) (field Inputs) New_Task_Unique_Identifier returns: %s", err.Error())
			return
		}
		var preTaskStr string
		preTaskStr, err = preId.String()
		if err != nil {
			err = fmt.Errorf("(ValidateDependants) (field Inputs) task.String returned: %s", err.Error())
			return
		}
		preTask, ok, xerr := qm.TaskMap.Get(preId, true)
		if xerr != nil {
			err = fmt.Errorf("(ValidateDependants) (field Inputs) predecessor task %s not found for task %s: %s", preTaskStr, task.ID, xerr.Error())
			return
		}
		if !ok {
			reason = fmt.Sprintf("(ValidateDependants) (field Inputs) predecessor task not found: task=%s, pretask=%s", task.ID, preTaskStr)
			logger.Debug(3, reason)
			return
		}
		var preTaskState string
		preTaskState, err = preTask.GetStateTimeout(time.Second * 1)
		if err != nil {
			err = nil
			skip = true
			//err = fmt.Errorf("(ValidateDependants) (field Inputs) unable to get state for predecessor task %s: %s", preTaskStr, xerr.Error())
			return
		}
		if preTaskState != TASK_STAT_COMPLETED {
			reason = fmt.Sprintf("(ValidateDependants) (field Inputs) predecessor task state is not completed: task=%s, pretask=%s, pretask.state=%s", task.ID, preTaskStr, preTaskState)
			logger.Debug(3, reason)
			return
		}
	}
	return
}

// ValidateInputs _
func (task *Task) ValidateInputs(qm *ServerMgr) (err error) {
	err = task.LockNamed("ValidateInputs")
	if err != nil {
		err = fmt.Errorf("(ValidateInputs) unable to lock task %s: %s", task.ID, err.Error())
		return
	}
	defer task.Unlock()

	for _, io := range task.Inputs {
		if io.Origin != "" {
			// find predecessor task
			var preId Task_Unique_Identifier
			preId, err = New_Task_Unique_Identifier(task.JobId, io.Origin)
			if err != nil {
				err = fmt.Errorf("(ValidateInputs) New_Task_Unique_Identifier returned: %s", err.Error())
				return
			}
			var preTaskStr string
			preTaskStr, err = preId.String()
			if err != nil {
				err = fmt.Errorf("(ValidateInputs) task.String returned: %s", err.Error())
				return
			}
			preTask, ok, xerr := qm.TaskMap.Get(preId, true)
			if xerr != nil {
				err = fmt.Errorf("(ValidateInputs) predecessor task %s not found for task %s: %s", preTaskStr, task.ID, xerr.Error())
				return
			}
			if !ok {
				err = fmt.Errorf("(ValidateInputs) predecessor task %s not found for task %s", preTaskStr, task.ID)
				return
			}

			// test predecessor state
			preTaskState, xerr := preTask.GetState()
			if xerr != nil {
				err = fmt.Errorf("(ValidateInputs) unable to get state for predecessor task %s: %s", preTaskStr, xerr.Error())
				return
			}
			if preTaskState != TASK_STAT_COMPLETED {
				err = fmt.Errorf("(ValidateInputs) predecessor task state is not completed: task=%s, pretask=%s, pretask.state=%s", task.ID, preTaskStr, preTaskState)
				return
			}

			// find predecessor output
			preTaskIO, xerr := preTask.GetOutput(io.FileName)
			if xerr != nil {
				err = fmt.Errorf("(ValidateInputs) unable to get IO for predecessor task %s, file %s: %s", preTaskStr, io.FileName, err.Error())
				return
			}

			io.Node = preTaskIO.Node
		}

		// make sure we have node id
		if (io.Node == "") || (io.Node == "-") {
			err = fmt.Errorf("(ValidateInputs) error in locate input for task, no node id found: task=%s, file=%s", task.ID, io.FileName)
			return
		}

		// force build data url
		io.Url = ""
		_, err = io.DataUrl()
		if err != nil {
			err = fmt.Errorf("(ValidateInputs) DataUrl returns: %s", err.Error())
			return
		}

		// forece check file exists and get size
		io.Size = 0
		_, err = io.UpdateFileSize()
		if err != nil {
			err = fmt.Errorf("(ValidateInputs) input file %s UpdateFileSize returns: %s", io.FileName, err.Error())
			return
		}

		// create or wait on shock index on input node (if set in workflow document)
		_, err = io.IndexFile(io.ShockIndex)
		if err != nil {
			err = fmt.Errorf("(ValidateInputs) failed to create shock index: task=%s, node=%s: %s", task.ID, io.Node, err.Error())
			return
		}

		logger.Debug(3, "(ValidateInputs) input located: task=%s, file=%s, node=%s, size=%d", task.ID, io.FileName, io.Node, io.Size)
	}

	if task.WorkflowInstanceID == "" {
		err = dbUpdateJobTaskIO(task.JobId, task.WorkflowInstanceID, task.ID, "inputs", task.Inputs)
		if err != nil {
			err = fmt.Errorf("(ValidateInputs) unable to save task inputs to mongodb, task=%s: %s", task.ID, err.Error())
			return
		}
	} else {
		err = dbUpdateTaskIO(task.WorkflowInstanceUUID, task.ID, "inputs", task.Inputs)
		if err != nil {
			err = fmt.Errorf("(ValidateInputs) unable to save task inputs to mongodb, task=%s: %s", task.ID, err.Error())
			return
		}
	}
	return
}

// ValidateOutputs _
func (task *Task) ValidateOutputs() (err error) {
	err = task.LockNamed("ValidateOutputs")
	if err != nil {
		err = fmt.Errorf("(ValidateOutputs) unable to lock task %s: %s", task.ID, err.Error())
		return
	}
	defer task.Unlock()

	for _, io := range task.Outputs {

		// force build data url
		io.Url = ""
		_, err = io.DataUrl()
		if err != nil {
			err = fmt.Errorf("(ValidateOutputs) DataUrl returns: %s", err.Error())
			return
		}

		// force check file exists and get size
		io.Size = 0
		_, err = io.UpdateFileSize()
		if err != nil {
			err = fmt.Errorf("(ValidateOutputs) input file %s GetFileSize returns: %s", io.FileName, err.Error())
			return
		}

		// create or wait on shock index on output node (if set in workflow document)
		_, err = io.IndexFile(io.ShockIndex)
		if err != nil {
			err = fmt.Errorf("(ValidateOutputs) failed to create shock index: task=%s, node=%s: %s", task.ID, io.Node, err.Error())
			return
		}
	}

	if task.WorkflowInstanceID == "" {
		err = dbUpdateJobTaskIO(task.JobId, task.WorkflowInstanceID, task.ID, "outputs", task.Outputs)
		if err != nil {
			err = fmt.Errorf("(ValidateOutputs) unable to save task outputs to mongodb, task=%s: %s", task.ID, err.Error())
			return
		}
	} else {
		err = dbUpdateTaskIO(task.WorkflowInstanceUUID, task.ID, "outputs", task.Outputs)
		if err != nil {
			err = fmt.Errorf("(ValidateOutputs) unable to save task outputs to mongodb, task=%s: %s", task.ID, err.Error())
			return
		}
	}
	return
}

// ValidatePredata _
func (task *Task) ValidatePredata() (err error) {
	err = task.LockNamed("ValidatePreData")
	if err != nil {
		err = fmt.Errorf("unable to lock task %s: %s", task.ID, err.Error())
		return
	}
	defer task.Unlock()

	// locate predata
	var modified bool
	for _, io := range task.Predata {
		// only verify predata that is a shock node
		if (io.Node != "") && (io.Node != "-") {
			// check file size
			mod, xerr := io.UpdateFileSize()
			if xerr != nil {
				err = fmt.Errorf("input file %s GetFileSize returns: %s", io.FileName, xerr.Error())
				return
			}
			if mod {
				modified = true
			}
			// build url if missing
			if io.Url == "" {
				_, err = io.DataUrl()
				if err != nil {
					err = fmt.Errorf("DataUrl returns: %s", err.Error())
				}
				modified = true
			}
		}
	}

	if modified {
		if task.WorkflowInstanceID == "" {
			err = dbUpdateJobTaskIO(task.JobId, task.WorkflowInstanceID, task.ID, "predata", task.Predata)
			if err != nil {
				err = fmt.Errorf("unable to save task predata to mongodb, task=%s: %s", task.ID, err.Error())
			}
		} else {
			err = dbUpdateTaskIO(task.WorkflowInstanceUUID, task.ID, "predata", task.Predata)
			if err != nil {
				err = fmt.Errorf("unable to save task predata to mongodb, task=%s: %s", task.ID, err.Error())
			}
		}
	}
	return
}

// DeleteOutput _
func (task *Task) DeleteOutput() (modified int) {
	modified = 0
	taskState := task.State
	if taskState == TASK_STAT_COMPLETED ||
		taskState == TASK_STAT_SKIPPED ||
		taskState == TASK_STAT_FAIL_SKIP {
		for _, io := range task.Outputs {
			if io.Delete {
				if err := io.DeleteNode(); err != nil {
					logger.Warning("failed to delete shock node %s: %s", io.Node, err.Error())
				}
				modified++
			}
		}
	}
	return
}

// DeleteInput _
func (task *Task) DeleteInput() (modified int) {
	modified = 0
	taskState := task.State
	if taskState == TASK_STAT_COMPLETED ||
		taskState == TASK_STAT_SKIPPED ||
		taskState == TASK_STAT_FAIL_SKIP {
		for _, io := range task.Inputs {
			if io.Delete {
				if err := io.DeleteNode(); err != nil {
					logger.Warning("failed to delete shock node %s: %s", io.Node, err.Error())
				}
				modified++
			}
		}
	}
	return
}

// DeleteLogs _
func (task *Task) DeleteLogs(logname string, writelock bool) (err error) {
	if writelock {
		err = task.LockNamed("setTotalWork")
		if err != nil {
			return
		}
		defer task.Unlock()
	}

	var logdir string
	logdir, err = getPathByJobID(task.JobId)
	if err != nil {
		err = fmt.Errorf("(task/GetTaDeleteLogsskLogs) getPathByJobId returned: %s", err.Error())
		return
	}
	globpath := fmt.Sprintf("%s/%s_*.%s", logdir, task.ID, logname)

	var logfiles []string
	logfiles, err = filepath.Glob(globpath)
	if err != nil {
		err = fmt.Errorf("(task/GetTaDeleteLogsskLogs) filepath.Glob returned: %s", err.Error())
		return
	}

	for _, logfile := range logfiles {
		workid := strings.Split(filepath.Base(logfile), ".")[0]
		logger.Debug(2, "Deleted %s log for workunit %s", logname, workid)
		os.Remove(logfile)
	}
	return
}

// GetStepOutput _
func (task *Task) GetStepOutput(name string) (obj cwl.CWLType, ok bool, reason string, err error) {

	if task.StepOutput == nil {
		ok = false
		reason = "task.StepOutput == nil"
		//err = fmt.Errorf("(task/GetStepOutput) task.StepOutput == nil")
		return
	}

	outputList := ""
	for _, namedStepOutput := range *task.StepOutput {

		namedStepOutputBase := path.Base(namedStepOutput.ID)
		outputList += "," + namedStepOutputBase

		logger.Debug(3, "(task/GetStepOutput) %s vs %s\n", namedStepOutputBase, name)
		if namedStepOutputBase == name {

			obj = namedStepOutput.Value

			if obj == nil {
				err = fmt.Errorf("(task/GetStepOutput) found %s , but it is nil", name) // this should not happen, taskReady makes sure everything is available
				return
			}

			ok = true

			return

		}

	}

	if outputList == "" {
		outputList = "nothing"
	}

	reason = fmt.Sprintf("(GetStepOutput) task=%s  only found: %s", task.ID, outputList)
	ok = false
	return
}

// GetStepOutputNames _
func (task *Task) GetStepOutputNames() (names []string, err error) {

	if task.StepOutput == nil {
		err = fmt.Errorf("(task/GetStepOutputNames) task.StepOutput == nil")
		return
	}

	names = []string{}

	for _, namedStepOutput := range *task.StepOutput {

		namedStepOutputBase := path.Base(namedStepOutput.ID)

		names = append(names, namedStepOutputBase)

	}

	return
}

// String2Date _
func String2Date(str string) (t time.Time, err error) {
	//layout := "2006-01-02T15:04:05.00Z"
	// 2018-12-13T22:36:02.96Z
	//str := "2014-11-12T11:45:26.371Z"
	//t, err = time.Parse(layout, str)
	t, err = time.Parse(time.RFC3339, str)

	return
}

// FixTimeInMap _
func FixTimeInMap(originalMap map[string]interface{}, field string) (err error) {
	var value_if interface{}
	var ok bool
	value_if, ok = originalMap[field]
	if ok {
		switch value_if.(type) {
		case string:
			value_str := value_if.(string)

			var value_time time.Time
			value_time, err = String2Date(value_str)
			if err != nil {
				err = fmt.Errorf("(FixTimeInMap) Could not parse date: %s", err.Error())
				return
			}
			delete(originalMap, field)
			originalMap[field] = value_time

		case time.Time:
			// all ok
		default:
			err = fmt.Errorf("(FixTimeInMap) time type unknown (%s)", reflect.TypeOf(value_if))
		}

	}
	return
}

// NewTaskFromInterface _
func NewTaskFromInterface(original interface{}, context *cwl.WorkflowContext) (task *Task, err error) {

	task = &Task{}
	task.TaskRaw = TaskRaw{}

	//spew.Dump(original)

	original, err = cwl.MakeStringMap(original, context)
	if err != nil {
		err = fmt.Errorf("(NewTaskFromInterface) MakeStringMap returned: %s", err.Error())
		return
	}

	originalMap := original.(map[string]interface{})

	for _, field := range []string{"createdDate", "startedDate", "completedDate"} {

		err = FixTimeInMap(originalMap, field)
		if err != nil {
			err = fmt.Errorf("(NewTaskFromInterface) FixTimeInMap returned: %s", err.Error())
			return
		}
	}

	err = mapstructure.Decode(original, task)
	if err != nil {
		err = fmt.Errorf("(NewTaskFromInterface) mapstructure.Decode returned: %s (%s)", err.Error(), spew.Sdump(originalMap))
		return
	}

	if task.WorkflowInstanceID == "" {
		err = fmt.Errorf("(NewTaskFromInterface) task.WorkflowInstanceID == empty")
		return
	}

	// if task.WorkflowInstanceID != "_root" {
	// 	if task.WorkflowParent == nil {
	// 		task_id_str, _ := task.String()
	// 		err = fmt.Errorf("(NewTaskFromInterface) task.WorkflowParent == nil , (%s)", task_id_str)
	// 		return
	// 	}
	// }
	return
}

// NewTasksFromInterface _
func NewTasksFromInterface(original interface{}, context *cwl.WorkflowContext) (tasks []*Task, err error) {

	switch original.(type) {
	case []interface{}:
		originalArray := original.([]interface{})

		tasks = []*Task{}

		for i := range originalArray {

			var t *Task
			t, err = NewTaskFromInterface(originalArray[i], context)
			if err != nil {
				err = fmt.Errorf("(NewTasksFromInterface) NewTaskFromInterface returned: %s", err.Error())
				return
			}

			tasks = append(tasks, t)

		}

	default:
		err = fmt.Errorf("(NewTasksFromInterface) type not supported: %s", reflect.TypeOf(original))
	}

	return
}
