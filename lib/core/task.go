package core

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/logger"
	"strings"
	"time"
)

const (
	TASK_STAT_INIT             = "init"
	TASK_STAT_QUEUED           = "queued"
	TASK_STAT_INPROGRESS       = "in-progress"
	TASK_STAT_PENDING          = "pending"
	TASK_STAT_SUSPEND          = "suspend"
	TASK_STAT_FAILED           = "failed"
	TASK_STAT_FAILED_PERMANENT = "failed-permanent"
	TASK_STAT_COMPLETED        = "completed"
	TASK_STAT_SKIPPED          = "user_skipped"
	TASK_STAT_FAIL_SKIP        = "skipped"
	TASK_STAT_PASSED           = "passed"
)

type TaskRaw struct {
	RWMutex
	Id          string    `bson:"taskid" json:"taskid"`
	JobId       string    `bson:"jobid" json:"jobid"`
	Info        *Info     `bson:"-" json:"-"`
	Cmd         *Command  `bson:"cmd" json:"cmd"`
	Partition   *PartInfo `bson:"partinfo" json:"-"`
	DependsOn   []string  `bson:"dependsOn" json:"dependsOn"` // only needed if dependency cannot be inferred from Input.Origin
	TotalWork   int       `bson:"totalwork" json:"totalwork"`
	MaxWorkSize int       `bson:"maxworksize"   json:"maxworksize"`
	RemainWork  int       `bson:"remainwork" json:"remainwork"`
	//WorkStatus  []string  `bson:"workstatus" json:"-"`
	State string `bson:"state" json:"state"`
	//Skip          int               `bson:"skip" json:"-"`
	CreatedDate   time.Time         `bson:"createdDate" json:"createddate"`
	StartedDate   time.Time         `bson:"startedDate" json:"starteddate"`
	CompletedDate time.Time         `bson:"completedDate" json:"completeddate"`
	ComputeTime   int               `bson:"computetime" json:"computetime"`
	UserAttr      map[string]string `bson:"userattr" json:"userattr"`
	ClientGroups  string            `bson:"clientgroups" json:"clientgroups"`
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

func NewTaskRaw(task_id string, info *Info) TaskRaw {

	logger.Debug(3, "task_id: %s", task_id)

	return TaskRaw{
		Id:        task_id,
		Info:      info,
		Cmd:       &Command{},
		Partition: nil,
		DependsOn: []string{},
	}
}

func (task *TaskRaw) InitRaw(job *Job) (changed bool, err error) {
	changed = false

	if len(task.Id) == 0 {
		err = errors.New("(TaskRaw.InitRaw) empty taskid")
		return
	}

	task.RWMutex.Init("task_" + task.Id)

	job_id := job.Id

	if job_id == "" {
		err = fmt.Errorf("(NewTask) job_id empty")
		return
	}
	task.JobId = job_id

	if task.State == "" {
		task.State = TASK_STAT_INIT
	}

	if !strings.Contains(task.Id, "_") {
		// is not standard taskid, convert it
		task.Id = fmt.Sprintf("%s_%s", job.Id, task.Id)
		changed = true
	}

	fix_DependsOn := false
	for _, dependency := range task.DependsOn {
		if !strings.Contains(dependency, "_") {
			fix_DependsOn = true
		}
	}

	if fix_DependsOn {
		changed = true
		new_DependsOn := []string{}
		for _, dependency := range task.DependsOn {
			if strings.Contains(dependency, "_") {
				new_DependsOn = append(new_DependsOn, dependency)
			} else {
				new_DependsOn = append(new_DependsOn, fmt.Sprintf("%s_%s", job.Id, dependency))
			}
		}
		task.DependsOn = new_DependsOn
	}

	if job.Info == nil {
		err = fmt.Errorf("(NewTask) job.Info empty")
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

	return
}

func (task *Task) Init(job *Job) (changed bool, err error) {
	changed, err = task.InitRaw(job)
	if err != nil {
		return
	}

	// populate DependsOn
	deps := make(map[string]bool)
	deps_changed := false
	// collect explicit dependencies
	for _, deptask := range task.DependsOn {
		if !strings.Contains(deptask, "_") {
			err = fmt.Errorf("deptask \"%s\" is missing _", deptask)
			return
		}
		deps[deptask] = true
	}

	for _, input := range task.Inputs {
		if input.Origin != "" {
			origin := input.Origin
			if !strings.Contains(origin, "_") {
				origin = fmt.Sprintf("%s_%s", job.Id, origin)
			}
			_, ok := deps[origin]
			if !ok {
				// this was not yet in deps
				deps[origin] = true
				deps_changed = true
			}
		}
	}

	// write all dependencies if different from before
	if deps_changed {
		task.DependsOn = []string{}
		for deptask, _ := range deps {
			task.DependsOn = append(task.DependsOn, deptask)
		}
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
func NewTask(job *Job, task_id string) (t *Task, err error) {
	t = &Task{
		TaskRaw: NewTaskRaw(task_id, job.Info),
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

func (task *TaskRaw) GetState() (state string, err error) {
	lock, err := task.RLockNamed("GetState")
	if err != nil {
		return
	}
	defer task.RUnlockNamed(lock)
	state = task.State
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

func (task *TaskRaw) GetId() (id string, err error) {
	lock, err := task.RLockNamed("GetId")
	if err != nil {
		return
	}
	defer task.RUnlockNamed(lock)
	id = task.Id
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

func (task *TaskRaw) SetState(new_state string) (err error) {
	err = task.LockNamed("SetState")
	if err != nil {
		return
	}
	defer task.Unlock()

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
				err = ShockPutIndex(io.Host, io.Node, io.ShockIndex, task.Info.DataToken)
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
		err = ShockPutIndex(inputIO.Host, inputIO.Node, newPartition.Index, task.Info.DataToken)
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

func (task *Task) CreateWorkunits() (wus []*Workunit, err error) {
	//if a task contains only one workunit, assign rank 0
	if task.TotalWork == 1 {
		workunit := NewWorkunit(task, 0)
		wus = append(wus, workunit)
		return
	}
	// if a task contains N (N>1) workunits, assign rank 1..N
	for i := 1; i <= task.TotalWork; i++ {
		workunit := NewWorkunit(task, i)
		wus = append(wus, workunit)
	}
	return
}

func (task *Task) GetTaskLogs() (tlog *TaskLog) {
	tlog = new(TaskLog)
	tlog.Id = task.Id
	tlog.State = task.State
	tlog.TotalWork = task.TotalWork
	tlog.CompletedDate = task.CompletedDate
	if task.TotalWork == 1 {
		tlog.Workunits = append(tlog.Workunits, NewWorkLog(task.Id, 0))
	} else {
		for i := 1; i <= task.TotalWork; i++ {
			tlog.Workunits = append(tlog.Workunits, NewWorkLog(task.Id, i))
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
