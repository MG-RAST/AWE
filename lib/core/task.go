package core

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/logger"
	"os/exec"
	"strings"
	"time"
)

const (
	TASK_STAT_INIT       = "init"
	TASK_STAT_QUEUED     = "queued"
	TASK_STAT_INPROGRESS = "in-progress"
	TASK_STAT_PENDING    = "pending"
	TASK_STAT_SUSPEND    = "suspend"
	TASK_STAT_COMPLETED  = "completed"
	TASK_STAT_SKIPPED    = "user_skipped"
	TASK_STAT_FAIL_SKIP  = "skipped"
	TASK_STAT_PASSED     = "passed"
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
	WorkStatus  []string  `bson:"workstatus" json:"-"`
	State       string    `bson:"state" json:"state"`
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

func NewTaskRaw(job_id string, task_id string, info *Info) TaskRaw {

	logger.Debug(0, "Task.Id: %s_%s", job_id, task_id)

	return TaskRaw{
		//Id:         fmt.Sprintf("%s_%s", job_id, task_id),
		Id:         task_id,
		Info:       info,
		Cmd:        &Command{},
		Partition:  nil,
		DependsOn:  []string{},
		TotalWork:  1,
		RemainWork: 1,
		WorkStatus: []string{},
		State:      TASK_STAT_INIT,
		//Skip:       0,
	}
}

func (task *TaskRaw) Init() {
	task.RWMutex.Init("task_" + task.Id)
}

func NewTask(job *Job, task_id string) (t *Task, err error) {

	job_id := job.Id
	if job_id == "" {
		err = fmt.Errorf("(NewTask) job_id empty")
		return
	}

	t = &Task{
		TaskRaw: NewTaskRaw(job_id, task_id, job.Info),
		Inputs:  []*IO{},
		Outputs: []*IO{},
		Predata: []*IO{},
	}
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

func (task *TaskRaw) SetState(new_state string) (err error) {
	err = task.LockNamed("SetState")
	if err != nil {
		return
	}
	defer task.Unlock()

	if task.State == new_state {
		return
	}
	if new_state == TASK_STAT_COMPLETED {
		if task.State != TASK_STAT_COMPLETED {

			// state TASK_STAT_COMPLETED is new!
			err = dbIncrementJobField(task.JobId, "remaintasks", -1)
			if err != nil {
				return
			}

			this_time := time.Now()
			task.CompletedDate = this_time
			dbUpdateJobTaskField(task.JobId, task.Id, "completeddate", this_time)
		}

	} else {
		// in case a completed teask is marked as something different
		if task.State == TASK_STAT_COMPLETED {
			err = dbIncrementJobField(task.JobId, "remaintasks", 1)
			if err != nil {
				return
			}
		}

	}
	dbUpdateJobTaskField(task.JobId, task.Id, "state", new_state)
	task.State = new_state
	return
}

func (task *TaskRaw) SetCompletedDate_DEPRECATED(date time.Time) (err error) {
	err = task.LockNamed("SetCompletedDate")
	if err != nil {
		return
	}
	defer task.Unlock()
	task.CompletedDate = date
	return
}

//func (task *TaskRaw) GetSkip() (skip int, err error) {
//	lock, err := task.RLockNamed("GetSkip")
//	if err != nil {
//		return
//	}
//	defer task.RUnlockNamed(lock)
//	skip = task.Skip
//	return
//}

func (task *TaskRaw) GetDependsOn() (dep []string, err error) {
	lock, err := task.RLockNamed("GetDependsOn")
	if err != nil {
		return
	}
	defer task.RUnlockNamed(lock)
	dep = task.DependsOn
	return
}

// fill some info (lacked in input json) for a task
func (task *Task) InitTask(job *Job) (err error) {
	//validate taskid
	if len(task.Id) == 0 {
		return errors.New("invalid taskid:" + task.Id)
	}

	parts := strings.Split(task.Id, "_")
	if len(parts) == 1 {
		// is not standard taskid, convert it
		task.Id = fmt.Sprintf("%s_%s", job.Id, task.Id)
		for j := 0; j < len(task.DependsOn); j++ {
			depend := task.DependsOn[j]
			task.DependsOn[j] = fmt.Sprintf("%s_%s", job.Id, depend)
		}
	}

	task.Info = job.Info

	if task.TotalWork <= 0 {
		task.setTotalWork(1)
	}
	task.WorkStatus = make([]string, task.TotalWork)
	task.RemainWork = task.TotalWork

	// set node / host / url for files
	for _, io := range task.Inputs {
		if io.Node == "" {
			io.Node = "-"
		}
		if _, err = io.DataUrl(); err != nil {
			return err
		}
		logger.Debug(2, "inittask input: host="+io.Host+", node="+io.Node+", url="+io.Url)
	}
	for _, io := range task.Outputs {
		if io.Node == "" {
			io.Node = "-"
		}
		if _, err = io.DataUrl(); err != nil {
			return err
		}
		logger.Debug(2, "inittask output: host="+io.Host+", node="+io.Node+", url="+io.Url)
	}
	for _, io := range task.Predata {
		if io.Node == "" {
			io.Node = "-"
		}
		if _, err = io.DataUrl(); err != nil {
			return err
		}
		// predata IO can not be empty
		if (io.Url == "") && (io.Node == "-") {
			return errors.New("Invalid IO, required fields url or host / node missing")
		}
		logger.Debug(2, "inittask predata: host="+io.Host+", node="+io.Node+", url="+io.Url)
	}

	if len(task.Cmd.Environ.Private) > 0 {
		task.Cmd.HasPrivateEnv = true
	}

	task.setTokenForIO()
	err = task.SetState(TASK_STAT_INIT)

	return
}

func (task *Task) UpdateState(newState string) string {
	task.LockNamed("UpdateState")
	defer task.Unlock()
	task.State = newState
	return task.State
}

func (task *Task) CreateIndex() (err error) {
	for _, io := range task.Inputs {
		if len(io.ShockIndex) > 0 {
			idxinfo, err := io.GetIndexInfo()
			if err != nil {
				errMsg := "could not retrieve index info from input shock node, taskid=" + task.Id + ", error=" + err.Error()
				logger.Error("error: " + errMsg)
				return errors.New(errMsg)
			}

			if _, ok := idxinfo[io.ShockIndex]; !ok {
				if err := ShockPutIndex(io.Host, io.Node, io.ShockIndex, task.Info.DataToken); err != nil {
					errMsg := "failed to create index on shock node for taskid=" + task.Id + ", error=" + err.Error()
					logger.Error("error: " + errMsg)
					return errors.New(errMsg)
				}
			}
		}
	}
	return
}

//get part size based on partition/index info
//if fail to get index info, task.TotalWork fall back to 1 and return nil
func (task *Task) InitPartIndex() (err error) {
	if task.TotalWork == 1 && task.MaxWorkSize == 0 {
		return
	}
	var input_io *IO
	if task.Partition == nil {
		if len(task.Inputs) == 1 {
			for _, io := range task.Inputs {
				input_io = io
				task.Partition = new(PartInfo)
				task.Partition.Input = io.FileName
				task.Partition.MaxPartSizeMB = task.MaxWorkSize
				break
			}
		} else {
			task.setTotalWork(1)
			logger.Error("warning: lacking parition info while multiple inputs are specified, taskid=" + task.Id)
			return
		}
	} else {
		if task.MaxWorkSize > 0 {
			task.Partition.MaxPartSizeMB = task.MaxWorkSize
		}
		if task.Partition.MaxPartSizeMB == 0 && task.TotalWork <= 1 {
			task.setTotalWork(1)
			return
		}
		found := false
		for _, io := range task.Inputs {
			if io.FileName == task.Partition.Input {
				found = true
				input_io = io
			}
		}
		if !found {
			task.setTotalWork(1)
			logger.Error("warning: invalid partition info, taskid=" + task.Id)
			return
		}
	}

	var totalunits int

	idxinfo, err := input_io.GetIndexInfo()
	if err != nil {
		task.setTotalWork(1)
		logger.Error("warning: invalid file info, taskid=" + task.Id + ", error=" + err.Error())
		return nil
	}

	idxtype := conf.DEFAULT_INDEX
	if _, ok := idxinfo[idxtype]; !ok { //if index not available, create index
		if err := ShockPutIndex(input_io.Host, input_io.Node, idxtype, task.Info.DataToken); err != nil {
			task.setTotalWork(1)
			logger.Error("warning: fail to create index on shock for taskid=" + task.Id + ", error=" + err.Error())
			return nil
		}
		totalunits, err = input_io.TotalUnits(idxtype) //get index info again
		if err != nil {
			task.setTotalWork(1)
			logger.Error("warning: fail to get index units, taskid=" + task.Id + ", error=" + err.Error())
			return nil
		}
	} else { //index existing, use it directly
		totalunits = int(idxinfo[idxtype].TotalUnits)
	}

	//adjust total work based on needs
	if task.Partition.MaxPartSizeMB > 0 { // fixed max part size
		//this implementation for chunkrecord indexer only
		chunkmb := int(conf.DEFAULT_CHUNK_SIZE / 1048576)
		var totalwork int
		if totalunits*chunkmb%task.Partition.MaxPartSizeMB == 0 {
			totalwork = totalunits * chunkmb / task.Partition.MaxPartSizeMB
		} else {
			totalwork = totalunits*chunkmb/task.Partition.MaxPartSizeMB + 1
		}
		if totalwork < task.TotalWork { //use bigger splits (specified by size or totalwork)
			totalwork = task.TotalWork
		}
		task.setTotalWork(totalwork)
	}
	if totalunits < task.TotalWork {
		task.setTotalWork(totalunits)
	}

	task.Partition.Index = idxtype
	task.Partition.TotalIndex = totalunits
	return
}

func (task *Task) setTotalWork(num int) {
	task.TotalWork = num
	task.RemainWork = num
	task.WorkStatus = make([]string, num)
}

func (task *Task) SetRemainWork(num int) (err error) {
	err = task.LockNamed("SetRemainWork")
	if err != nil {
		return
	}
	defer task.Unlock()
	task.RemainWork = num

	return
}

func (task *Task) IncrementRemainWork(inc int) (remainwork int, err error) {
	err = task.LockNamed("IncrementRemainWork")
	if err != nil {
		return
	}
	defer task.Unlock()

	task.RemainWork += inc

	remainwork = task.RemainWork

	return
}

func (task *Task) IncrementComputeTime(inc_time int) (err error) {
	err = task.LockNamed("IncrementComputeTime")
	if err != nil {
		return
	}
	defer task.Unlock()

	task.ComputeTime += inc_time

	return
}

func (task *Task) setTokenForIO() {
	if !task.Info.Auth || task.Info.DataToken == "" {
		return
	}
	for _, io := range task.Inputs {
		io.DataToken = task.Info.DataToken
	}
	for _, io := range task.Outputs {
		io.DataToken = task.Info.DataToken
	}
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

//func (task *Task) Skippable() bool {
// For a task to be skippable, it should meet
// the following requirements (this may change
// in the future):
// 1.- It should have exactly one input file
// and one output file (This way, we can connect tasks
// Ti-1 and Ti+1 transparently)
// 2.- It should be a simple pipeline task. That is,
// it should just have at most one "parent" Ti-1 ---> Ti
//	return (len(task.Inputs) == 1) &&
//		(len(task.Outputs) == 1) &&
//		(len(task.DependsOn) <= 1)
//}

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

//creat index (=deprecated=)
func createIndex(host string, nodeid string, indexname string) (err error) {
	argv := []string{}
	argv = append(argv, "-X")
	argv = append(argv, "PUT")
	target_url := fmt.Sprintf("%s/node/%s?index=%s", host, nodeid, indexname)
	argv = append(argv, target_url)

	cmd := exec.Command("curl", argv...)
	err = cmd.Run()
	if err != nil {
		return
	}
	return
}
