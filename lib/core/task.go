package core

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/shock"
	"regexp"
	//"strconv"
	//"github.com/davecgh/go-spew/spew"
	"strings"
	"time"
)

const (
	TASK_STAT_INIT             = "init"        // initial state on creation of a task
	TASK_STAT_PENDING          = "pending"     // a task that wants to be enqueued
	TASK_STAT_READY            = "ready"       // a task ready to be enqueued
	TASK_STAT_QUEUED           = "queued"      // a task for which workunits have been created/queued
	TASK_STAT_INPROGRESS       = "in-progress" // a first workunit has been checkout (this does not guarantee a workunit is running right now)
	TASK_STAT_SUSPEND          = "suspend"
	TASK_STAT_FAILED           = "failed"
	TASK_STAT_FAILED_PERMANENT = "failed-permanent" // on exit code 42
	TASK_STAT_COMPLETED        = "completed"
	TASK_STAT_SKIPPED          = "user_skipped" // deprecated
	TASK_STAT_FAIL_SKIP        = "skipped"      // deprecated
	TASK_STAT_PASSED           = "passed"       // deprecated ?
)

const (
	TASK_TYPE_UNKNOWN  = ""
	TASK_TYPE_SCATTER  = "scatter"
	TASK_TYPE_WORKFLOW = "workflow"
	TASK_TYPE_NORMAL   = "normal"
)

type TaskRaw struct {
	RWMutex                `bson:"-" json:"-"`
	Task_Unique_Identifier `bson:",inline"`

	Id                  string                   `bson:"taskid" json:"taskid"` // old-style
	TaskType            string                   `bson:"task_type" json:"task_type"`
	Info                *Info                    `bson:"-" json:"-"` // this is just a pointer to the job.Info
	Cmd                 *Command                 `bson:"cmd" json:"cmd"`
	Partition           *PartInfo                `bson:"partinfo" json:"-"`
	DependsOn           []string                 `bson:"dependsOn" json:"dependsOn"` // only needed if dependency cannot be inferred from Input.Origin
	TotalWork           int                      `bson:"totalwork" json:"totalwork"`
	MaxWorkSize         int                      `bson:"maxworksize"   json:"maxworksize"`
	RemainWork          int                      `bson:"remainwork" json:"remainwork"`
	State               string                   `bson:"state" json:"state"`
	CreatedDate         time.Time                `bson:"createdDate" json:"createddate"`
	StartedDate         time.Time                `bson:"startedDate" json:"starteddate"`
	CompletedDate       time.Time                `bson:"completedDate" json:"completeddate"`
	ComputeTime         int                      `bson:"computetime" json:"computetime"`
	UserAttr            map[string]interface{}   `bson:"userattr" json:"userattr"`
	ClientGroups        string                   `bson:"clientgroups" json:"clientgroups"`
	WorkflowStep        *cwl.WorkflowStep        `bson:"workflowStep" json:"workflowStep"` // CWL-only
	StepOutputInterface interface{}              `bson:"stepOutput" json:"stepOutput"`     // CWL-only
	StepInput           *cwl.Job_document        `bson:"-" json:"-"`                       // CWL-only
	StepOutput          *cwl.Job_document        `bson:"-" json:"-"`                       // CWL-only
	Scatter_task        bool                     `bson:"scatter_task" json:"scatter_task"` // CWL-only
	Children            []Task_Unique_Identifier `bson:"children" json:"children"`         // CWL-only
	Children_ptr        []*Task                  `bson:"-" json:"-"`                       // CWL-only
	Finalizing          bool                     `bson:"-" json:"-"`                       // CWL-only
}

type Task struct {
	TaskRaw `bson:",inline"`
	Inputs  []*IO `bson:"inputs" json:"inputs"`
	Outputs []*IO `bson:"outputs" json:"outputs"`
	Predata []*IO `bson:"predata" json:"predata"`
}

// Deprecated JobDep struct uses deprecated TaskDep struct which uses the deprecated IOmap.  Maintained for backwards compatibility.
// Jobs that cannot be parsed into the Job struct, but can be parsed into the JobDep struct will be translated to the new Job struct.
// (=deprecated=)
type TaskDep struct {
	TaskRaw `bson:",inline"`
	Inputs  IOmap `bson:"inputs" json:"inputs"`
	Outputs IOmap `bson:"outputs" json:"outputs"`
	Predata IOmap `bson:"predata" json:"predata"`
}

type TaskLog struct {
	Id            string     `bson:"taskid" json:"taskid"`
	State         string     `bson:"state" json:"state"`
	TotalWork     int        `bson:"totalwork" json:"totalwork"`
	CompletedDate time.Time  `bson:"completedDate" json:"completeddate"`
	Workunits     []*WorkLog `bson:"workunits" json:"workunits"`
}

func NewTaskRaw(task_id Task_Unique_Identifier, info *Info) (tr TaskRaw, err error) {

	logger.Debug(3, "task_id: %s", task_id)
	logger.Debug(3, "task_id.JobId: %s", task_id.JobId)
	logger.Debug(3, "task_id.Parent: %s", task_id.Parent)
	logger.Debug(3, "task_id.TaskName: %s", task_id.TaskName)

	var task_str string
	task_str, err = task_id.String()
	if err != nil {
		err = fmt.Errorf("() task.String returned: %s", err.Error())
		return
	}

	tr = TaskRaw{
		Task_Unique_Identifier: task_id,
		Id:        task_str,
		Info:      info,
		Cmd:       &Command{},
		Partition: nil,
		DependsOn: []string{},
	}
	return
}

func (task *TaskRaw) InitRaw(job *Job) (changed bool, err error) {
	changed = false

	if len(task.Id) == 0 {
		err = errors.New("(InitRaw) empty taskid")
		return
	}

	job_id := job.Id

	if job_id == "" {
		err = fmt.Errorf("(InitRaw) job_id empty")
		return
	}

	if task.JobId == "" {
		task.JobId = job_id
		changed = true
	}

	//logger.Debug(3, "task.TaskName A: %s", task.TaskName)
	job_prefix := job_id + "_"
	if len(task.Id) > 0 && (!strings.HasPrefix(task.Id, job_prefix)) {
		task.TaskName = task.Id
		changed = true
	}
	//logger.Debug(3, "task.TaskName B: %s", task.TaskName)
	//if strings.HasSuffix(task.TaskName, "ERROR") {
	//	err = fmt.Errorf("(InitRaw) taskname is error")
	//	return
	//}

	if task.TaskName == "" && strings.HasPrefix(task.Id, job_prefix) {
		var tid Task_Unique_Identifier
		tid, err = New_Task_Unique_Identifier_FromString(task.Id)
		if err != nil {
			err = fmt.Errorf("(InitRaw) New_Task_Unique_Identifier_FromString returned: %s", err.Error())
			return
		}
		task.Task_Unique_Identifier = tid
	}

	var task_str string
	task_str, err = task.String()
	if err != nil {
		err = fmt.Errorf("(InitRaw) task.String returned: %s", err.Error())
		return
	}
	task.RWMutex.Init("task_" + task_str)

	// job_id is missing and task_id is only a number (e.g. on submission of old-style AWE)

	if task.TaskName == "" {
		err = fmt.Errorf("(InitRaw) task.TaskName empty")
		return
	}

	if task.Id != task_str {
		task.Id = task_str
		changed = true
	}

	if task.State == "" {
		task.State = TASK_STAT_INIT
		changed = true
	}

	if job.Info == nil {
		err = fmt.Errorf("(InitRaw) job.Info empty")
		return
	}
	task.Info = job.Info

	if task.TotalWork <= 0 {
		task.TotalWork = 1
	}

	if task.State != TASK_STAT_COMPLETED {
		if task.RemainWork != task.TotalWork {
			task.RemainWork = task.TotalWork
			changed = true
		}
	}

	if len(task.Cmd.Environ.Private) > 0 {
		task.Cmd.HasPrivateEnv = true
	}

	//if strings.HasPrefix(task.Id, task.JobId+"_") {
	//	task.Id = strings.TrimPrefix(task.Id, task.JobId+"_")
	//	changed = true
	//}

	//if strings.HasPrefix(task.Id, "_") {
	//	task.Id = strings.TrimPrefix(task.Id, "_")
	//	changed = true
	//}

	if task.StepOutputInterface != nil {
		task.StepOutput, err = cwl.NewJob_documentFromNamedTypes(task.StepOutputInterface)
		if err != nil {
			err = fmt.Errorf("(InitRaw) cwl.NewJob_documentFromNamedTypes returned: %s", err.Error())
			return
		}
	}

	return
}

// func (task *TaskRaw) String() (s string, err error) {
// 	err = task.LockNamed("String")
// 	if err != nil {
// 		return
// 	}
// 	defer task.Unlock()
//
// 	s = task.Task_Unique_Identifier.String()
//
// 	return
// }

// this function prevents a dead-lock when a sub-workflow task finalizes
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

func IsValidUUID(uuid string) bool {
	if len(uuid) != 36 {
		return false
	}
	r := regexp.MustCompile("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$")
	return r.MatchString(uuid)
}

// populate DependsOn
func (task *Task) CollectDependencies() (changed bool, err error) {

	deps := make(map[Task_Unique_Identifier]bool)
	deps_changed := false

	jobid, err := task.GetJobId()
	if err != nil {
		return
	}

	if jobid == "" {
		err = fmt.Errorf("(CollectDependencies) jobid is empty")
		return
	}

	job_prefix := jobid + "_"

	// collect explicit dependencies
	for _, deptask := range task.DependsOn {

		if deptask == "" {
			deps_changed = true
			continue
		}

		if !strings.HasPrefix(deptask, job_prefix) {
			deptask = job_prefix + deptask
			deps_changed = true
		}

		t, yerr := New_Task_Unique_Identifier_FromString(deptask)
		if yerr != nil {
			err = fmt.Errorf("(CollectDependencies) Cannot parse entry in DependsOn: %s", yerr.Error())
			return
		}

		if t.TaskName == "" {
			// this is to fix a bug
			deps_changed = true
			continue
		}

		deps[t] = true
	}

	for _, input := range task.Inputs {

		deptask := input.Origin
		if deptask == "" {
			deps_changed = true
			continue
		}

		if !strings.HasPrefix(deptask, job_prefix) {
			deptask = job_prefix + deptask
			deps_changed = true
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
			deps_changed = true
		}

	}

	// write all dependencies if different from before
	if deps_changed {
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

func (task *Task) Init(job *Job) (changed bool, err error) {
	changed, err = task.InitRaw(job)
	if err != nil {
		return
	}

	dep_changes, err := task.CollectDependencies()
	if err != nil {
		return
	}
	if dep_changes {
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

// currently this is only used to make a new task from a depricated task
func NewTask(job *Job, workflow string, task_id string) (t *Task, err error) {

	if job.Id == "" {
		err = fmt.Errorf("(NewTask) jobid is empty!")
		return
	}

	if strings.HasSuffix(workflow, "/") {
		err = fmt.Errorf("Suffix not in workflow_ids ok %s", task_id)
		return
	}

	if strings.HasSuffix(task_id, "/") {
		err = fmt.Errorf("Suffix in task_id not ok %s", task_id)
		return
	}

	task_id = strings.TrimSuffix(task_id, "/")
	workflow = strings.TrimSuffix(workflow, "/")

	var tui Task_Unique_Identifier
	tui, err = New_Task_Unique_Identifier(job.Id, workflow, task_id)
	if err != nil {
		return
	}

	var tr TaskRaw
	tr, err = NewTaskRaw(tui, job.Info)
	if err != nil {
		err = fmt.Errorf("(NewTask) NewTaskRaw returns: %s", err.Error())
		return
	}
	t = &Task{
		TaskRaw: tr,
		Inputs:  []*IO{},
		Outputs: []*IO{},
		Predata: []*IO{},
	}
	return
}

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

func (task *TaskRaw) GetChildren(qm *ServerMgr) (children []*Task, err error) {
	lock, err := task.RLockNamed("GetChildren")
	if err != nil {
		return
	}
	defer task.RUnlockNamed(lock)

	if task.Children_ptr == nil {
		children = []*Task{}
		for _, task_id := range task.Children {
			var child *Task
			var ok bool
			child, ok, err = qm.TaskMap.Get(task_id, true)
			if err != nil {
				return
			}
			if !ok {
				err = fmt.Errorf("(GetChildren) child task %s not found in TaskMap")
				return
			}
			children = append(children, child)
		}
		task.Children_ptr = children
	} else {
		children = task.Children_ptr
	}

	return
}

func (task *TaskRaw) GetParent() (p string, err error) {
	lock, err := task.RLockNamed("GetParent")
	if err != nil {
		return
	}
	defer task.RUnlockNamed(lock)
	p = task.Task_Unique_Identifier.Parent
	return
}

func (task *TaskRaw) GetState() (state string, err error) {
	lock, err := task.RLockNamed("GetState")
	if err != nil {
		return
	}
	defer task.RUnlockNamed(lock)
	state = task.State
	return
}

func (task *TaskRaw) GetTaskType() (type_str string, err error) {
	lock, err := task.RLockNamed("GetTaskType")
	if err != nil {
		return
	}
	defer task.RUnlockNamed(lock)
	type_str = task.TaskType
	return
}

func (task *Task) SetTaskType(type_str string, writelock bool) (err error) {
	if writelock {
		err = task.LockNamed("SetTaskType")
		if err != nil {
			return
		}
		defer task.Unlock()
	}
	err = dbUpdateJobTaskString(task.JobId, task.Id, "task_type", type_str)
	if err != nil {
		return
	}
	task.TaskType = type_str
	return
}

func (task *TaskRaw) SetCreatedDate(t time.Time) (err error) {
	err = task.LockNamed("SetCreatedDate")
	if err != nil {
		return
	}
	defer task.Unlock()

	err = dbUpdateJobTaskTime(task.JobId, task.Id, "createdDate", t)
	if err != nil {
		return
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

	err = dbUpdateJobTaskTime(task.JobId, task.Id, "startedDate", t)
	if err != nil {
		return
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

	err = dbUpdateJobTaskTime(task.JobId, task.Id, "completedDate", t)
	if err != nil {
		return
	}
	task.CompletedDate = t
	return
}

func (task *TaskRaw) SetStepOutput(jd *cwl.Job_document, lock bool) (err error) {
	if lock {
		err = task.LockNamed("SetStepOutput")
		if err != nil {
			return
		}
		defer task.Unlock()
	}

	err = dbUpdateJobTaskField(task.JobId, task.Id, "stepOutput", *jd)
	if err != nil {
		return
	}
	task.StepOutput = jd
	task.StepOutputInterface = jd
	return
}

// only for debugging purposes
func (task *TaskRaw) GetStateNamed(name string) (state string, err error) {
	lock, err := task.RLockNamed("GetState/" + name)
	if err != nil {
		return
	}
	defer task.RUnlockNamed(lock)
	state = task.State
	return
}

func (task *TaskRaw) GetId() (id Task_Unique_Identifier, err error) {
	lock, err := task.RLockNamed("GetId")
	if err != nil {
		return
	}
	defer task.RUnlockNamed(lock)
	id = task.Task_Unique_Identifier
	return
}

func (task *TaskRaw) GetJobId() (id string, err error) {
	lock, err := task.RLockNamed("GetJobId")
	if err != nil {
		return
	}
	defer task.RUnlockNamed(lock)
	id = task.JobId
	return
}

func (task *TaskRaw) SetState(new_state string, write_lock bool) (err error) {
	if write_lock {
		err = task.LockNamed("SetState")
		if err != nil {
			return
		}
		defer task.Unlock()
	}

	old_state := task.State
	taskid := task.Id
	jobid := task.JobId

	if jobid == "" {
		err = fmt.Errorf("task %s has no job id", taskid)
		return
	}
	if old_state == new_state {
		return
	}
	job, err := GetJob(jobid)
	if err != nil {
		return
	}

	err = dbUpdateJobTaskString(jobid, taskid, "state", new_state)
	if err != nil {
		return
	}

	logger.Debug(3, "(Task/SetState) %s new state: \"%s\" (old state \"%s\")", taskid, new_state, old_state)
	task.State = new_state

	if new_state == TASK_STAT_COMPLETED {
		err = job.IncrementRemainTasks(-1)
		if err != nil {
			return
		}
		err = task.SetCompletedDate(time.Now(), false)
		if err != nil {
			return
		}
	} else if old_state == TASK_STAT_COMPLETED {
		// in case a completed task is marked as something different
		err = job.IncrementRemainTasks(1)
		if err != nil {
			return
		}
		initTime := time.Time{}
		err = task.SetCompletedDate(initTime, false)
		if err != nil {
			return
		}
	}

	//r, err := dbGetJobTaskString(jobid, taskid, "state")
	//if err != nil {
	//	panic(err.Error())
	//}
	//if r != new_state {
	//	text := fmt.Sprintf("did set: \"%s\" got: \"%s\"", new_state, r)
	//	panic(text)
	//}

	//result_test, err := dbGetJobTask(jobid, taskid)

	//spew_config := spew.NewDefaultConfig()
	//spew_config.DisableMethods = true
	//spew_config.Dump(*result_test)

	return
}

func (task *TaskRaw) GetDependsOn() (dep []string, err error) {
	lock, err := task.RLockNamed("GetDependsOn")
	if err != nil {
		return
	}
	defer task.RUnlockNamed(lock)
	dep = task.DependsOn
	return
}

// checks and creates indices on shock node if needed
func (task *Task) CreateIndex() (err error) {
	for _, io := range task.Inputs {
		if len(io.ShockIndex) > 0 {
			_, hasIndex, err := io.GetIndexInfo(io.ShockIndex)
			if err != nil {
				errMsg := "could not retrieve index info from input shock node, taskid=" + task.Id + ", error=" + err.Error()
				logger.Error(errMsg)
				return errors.New(errMsg)
			}
			if !hasIndex {
				// create missing index
				sc := shock.ShockClient{Host: io.Host, Token: task.Info.DataToken}
				err = sc.ShockPutIndex(io.Node, io.ShockIndex)
				if err != nil {
					errMsg := "failed to create index on shock node for taskid=" + task.Id + ", error=" + err.Error()
					logger.Error("error: " + errMsg)
					return errors.New(errMsg)
				}
			}
		}
	}
	return
}

// get part size based on partition/index info
// this resets task.Partition when called
// only 1 task.Inputs allowed unless 'partinfo.input' specified on POST
// if fail to get index info, task.TotalWork set to 1 and task.Partition set to nil
func (task *Task) InitPartIndex() (err error) {
	if task.TotalWork == 1 && task.MaxWorkSize == 0 {
		// only 1 workunit requested
		return
	}

	err = task.LockNamed("InitPartIndex")
	if err != nil {
		return
	}
	defer task.Unlock()

	inputIO := task.Inputs[0]
	newPartition := &PartInfo{
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
			logger.Error("warning: lacking partition info while multiple inputs are specified, taskid=" + task.Id)
			err = task.setSingleWorkunit(false)
			return
		}
	}

	// if submitted with partition index use that, otherwise default
	if (task.Partition != nil) && (task.Partition.Index != "") {
		newPartition.Index = task.Partition.Index
	} else {
		newPartition.Index = conf.DEFAULT_INDEX
	}

	idxInfo, hasIndex, err := inputIO.GetIndexInfo(newPartition.Index)
	if err != nil {
		// bad state - set as not multi-workunit
		logger.Error("warning: invalid file info, taskid=%s, error=%s", task.Id, err.Error())
		err = task.setSingleWorkunit(false)
		return
	}

	var totalunits int
	if !hasIndex {
		// if index not available, create index
		sc := shock.ShockClient{Host: inputIO.Host, Token: task.Info.DataToken}
		err = sc.ShockPutIndex(inputIO.Node, newPartition.Index)
		if err != nil {
			// bad state - set as not multi-workunit
			logger.Error("warning: failed to create index %s on shock for taskid=%s, error=%s", newPartition.Index, task.Id, err.Error())
			err = task.setSingleWorkunit(false)
			return
		}
		totalunits, err = inputIO.TotalUnits(newPartition.Index) // get index info again
		if err != nil {
			// bad state - set as not multi-workunit
			logger.Error("warning: failed to get index %s units, taskid=%s, error=%s", newPartition.Index, task.Id, err.Error())
			err = task.setSingleWorkunit(false)
			return
		}
	} else {
		// index existing, use it directly
		totalunits = int(idxInfo.TotalUnits)
	}

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
		if totalwork != task.TotalWork {
			err = task.setTotalWork(totalwork, false)
			if err != nil {
				return
			}
		}
	}
	if totalunits < task.TotalWork {
		err = task.setTotalWork(totalunits, false)
		if err != nil {
			return
		}
	}

	// need only 1 workunit
	if task.TotalWork == 1 {
		err = task.setSingleWorkunit(false)
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
			return
		}
	}
	if task.Partition != nil {
		err = task.setPartition(nil, writelock)
		if err != nil {
			return
		}
	}
	if task.MaxWorkSize != 0 {
		err = task.setMaxWorkSize(0, writelock)
		if err != nil {
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
	err = dbUpdateJobTaskInt(task.JobId, task.Id, "totalwork", num)
	if err != nil {
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
	err = dbUpdateJobTaskPartition(task.JobId, task.Id, partition)
	if err != nil {
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
	err = dbUpdateJobTaskInt(task.JobId, task.Id, "maxworksize", num)
	if err != nil {
		return
	}
	task.MaxWorkSize = num
	return
}

func (task *Task) SetRemainWork(num int, writelock bool) (err error) {
	if writelock {
		err = task.LockNamed("SetRemainWork")
		if err != nil {
			return
		}
		defer task.Unlock()
	}
	err = dbUpdateJobTaskInt(task.JobId, task.Id, "remainwork", num)
	if err != nil {
		return
	}
	task.RemainWork = num
	return
}

// func (task *Task) SetSubworkflow(value string) (err error) {
//
// 	err = task.LockNamed("SetSubworkflow")
// 	if err != nil {
// 		return
// 	}
// 	defer task.Unlock()
//
// 	err = dbUpdateJobTaskString(task.JobId, task.Id, "subworkflow", value)
// 	if err != nil {
// 		return
// 	}
// 	task.Subworkflow = value
// 	return
// }

func (task *Task) IncrementRemainWork(inc int, writelock bool) (remainwork int, err error) {
	if writelock {
		err = task.LockNamed("IncrementRemainWork")
		if err != nil {
			return
		}
		defer task.Unlock()
	}

	remainwork = task.RemainWork + inc
	err = dbUpdateJobTaskInt(task.JobId, task.Id, "remainwork", remainwork)
	if err != nil {
		return
	}
	task.RemainWork = remainwork
	return
}

func (task *Task) IncrementComputeTime(inc int) (err error) {
	err = task.LockNamed("IncrementComputeTime")
	if err != nil {
		return
	}
	defer task.Unlock()

	newComputeTime := task.ComputeTime + inc
	err = dbUpdateJobTaskInt(task.JobId, task.Id, "computetime", newComputeTime)
	if err != nil {
		return
	}
	task.ComputeTime = newComputeTime
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
		err = dbUpdateJobTaskIO(task.JobId, task.Id, "inputs", task.Inputs)
		if err != nil {
			return
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
		err = dbUpdateJobTaskIO(task.JobId, task.Id, "outputs", task.Outputs)
	}
	return
}

func (task *Task) CreateWorkunits(qm *ServerMgr, job *Job) (wus []*Workunit, err error) {
	//if a task contains only one workunit, assign rank 0

	if task.TotalWork == 1 {
		workunit, xerr := NewWorkunit(qm, task, 0, job)
		if xerr != nil {
			err = fmt.Errorf("(CreateWorkunits) (single) NewWorkunit failed: %s", xerr.Error())
			return
		}
		wus = append(wus, workunit)
		return
	}
	// if a task contains N (N>1) workunits, assign rank 1..N
	for i := 1; i <= task.TotalWork; i++ {
		workunit, xerr := NewWorkunit(qm, task, i, job)
		if xerr != nil {
			err = fmt.Errorf("(CreateWorkunits) (multi) NewWorkunit failed: %s", xerr.Error())
			return
		}
		wus = append(wus, workunit)
	}
	return
}

func (task *Task) GetTaskLogs() (tlog *TaskLog, err error) {
	tlog = new(TaskLog)
	tlog.Id = task.Id
	tlog.State = task.State
	tlog.TotalWork = task.TotalWork
	tlog.CompletedDate = task.CompletedDate

	workunit_id := New_Workunit_Unique_Identifier(task.Task_Unique_Identifier, 0)
	//workunit_id := Workunit_Unique_Identifier{JobId: task.JobId, TaskName: task.Id}

	if task.TotalWork == 1 {
		//workunit_id.Rank = 0
		var wl *WorkLog
		wl, err = NewWorkLog(workunit_id)
		if err != nil {
			return
		}
		tlog.Workunits = append(tlog.Workunits, wl)
	} else {
		for i := 1; i <= task.TotalWork; i++ {
			workunit_id.Rank = i
			var wl *WorkLog
			wl, err = NewWorkLog(workunit_id)
			if err != nil {
				return
			}
			tlog.Workunits = append(tlog.Workunits, wl)
		}
	}
	return
}

func (task *Task) DeleteOutput() (modified int) {
	modified = 0
	task_state := task.State
	if task_state == TASK_STAT_COMPLETED ||
		task_state == TASK_STAT_SKIPPED ||
		task_state == TASK_STAT_FAIL_SKIP {
		for _, io := range task.Outputs {
			if io.Delete {
				if err := io.DeleteNode(); err != nil {
					logger.Warning("failed to delete shock node %s: %s", io.Node, err.Error())
				}
				modified += 1
			}
		}
	}
	return
}

func (task *Task) DeleteInput() (modified int) {
	modified = 0
	task_state := task.State
	if task_state == TASK_STAT_COMPLETED ||
		task_state == TASK_STAT_SKIPPED ||
		task_state == TASK_STAT_FAIL_SKIP {
		for _, io := range task.Inputs {
			if io.Delete {
				if err := io.DeleteNode(); err != nil {
					logger.Warning("failed to delete shock node %s: %s", io.Node, err.Error())
				}
				modified += 1
			}
		}
	}
	return
}

func (task *Task) UpdateInputs() (err error) {
	lock, err := task.RLockNamed("UpdateInputs")
	if err != nil {
		return
	}
	defer task.RUnlockNamed(lock)
	err = dbUpdateJobTaskIO(task.JobId, task.Id, "inputs", task.Inputs)
	return
}

func (task *Task) UpdateOutputs() (err error) {
	lock, err := task.RLockNamed("UpdateOutputs")
	if err != nil {
		return
	}
	defer task.RUnlockNamed(lock)
	err = dbUpdateJobTaskIO(task.JobId, task.Id, "outputs", task.Outputs)
	return
}

func (task *Task) UpdatePredata() (err error) {
	lock, err := task.RLockNamed("UpdatePredata")
	if err != nil {
		return
	}
	defer task.RUnlockNamed(lock)
	err = dbUpdateJobTaskIO(task.JobId, task.Id, "predata", task.Predata)
	return
}
