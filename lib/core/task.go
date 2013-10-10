package core

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/logger"
	"os/exec"
	"strconv"
	"strings"
)

const (
	TASK_STAT_INIT      = "init"
	TASK_STAT_QUEUED    = "queued"
	TASK_STAT_PENDING   = "pending"
	TASK_STAT_SUSPEND   = "suspend"
	TASK_STAT_COMPLETED = "completed"
	TASK_STAT_SKIPPED   = "user_skipped"
	TASK_STAT_FAIL_SKIP = "skipped"
	TASK_STAT_PASSED    = "passed"
)

type Task struct {
	Id          string    `bson:"taskid" json:"taskid"`
	Info        *Info     `bson:"info" json:"-"`
	Inputs      IOmap     `bson:"inputs" json:"inputs"`
	Outputs     IOmap     `bson:"outputs" json:"outputs"`
	Predata     IOmap     `bson:"predata" json:"predata"`
	Cmd         *Command  `bson:"cmd" json:"cmd"`
	Partition   *PartInfo `bson:"partinfo" json:"-"`
	DependsOn   []string  `bson:"dependsOn" json:"dependsOn"`
	TotalWork   int       `bson:"totalwork" json:"totalwork"`
	MaxWorkSize int       `bson:"maxworksize"   json:"maxworksize"`
	RemainWork  int       `bson:"remainwork" json:"remainwork"`
	WorkStatus  []string  `bson:"workstatus" json:"-"`
	State       string    `bson:"state" json:"state"`
	Skip        int       `bson:"skip" json:"-"`
}

func NewTask(job *Job, rank int) *Task {
	return &Task{
		Id:         fmt.Sprintf("%s_%d", job.Id, rank),
		Info:       job.Info,
		Inputs:     NewIOmap(),
		Outputs:    NewIOmap(),
		Cmd:        &Command{},
		Partition:  nil,
		DependsOn:  []string{},
		TotalWork:  1,
		RemainWork: 1,
		WorkStatus: []string{},
		State:      TASK_STAT_INIT,
		Skip:       0,
	}
}

// fill some info (lacked in input json) for a task
func (task *Task) InitTask(job *Job, rank int) (err error) {
	//validate taskid
	if len(task.Id) == 0 {
		return errors.New("invalid taskid:" + task.Id)
	}
	parts := strings.Split(task.Id, "_")
	if len(parts) == 2 {
		//is standard taskid (%s_%d), do nothing
	} else if idInt, err := strconv.Atoi(task.Id); err == nil {
		//if task.Id is an "integer", it is unmashalled from job.json (submitted by template)
		//convert to standard taskid
		if rank != idInt {
			return errors.New(fmt.Sprintf("invalid job script: task id doen't match stage %d vs %d", rank, idInt))
		}
		task.Id = fmt.Sprintf("%s_%s", job.Id, task.Id)
		for j := 0; j < len(task.DependsOn); j++ {
			depend := task.DependsOn[j]
			task.DependsOn[j] = fmt.Sprintf("%s_%s", job.Id, depend)
		}
	} else {
		return errors.New("invalid taskid:" + task.Id)
	}

	task.Info = job.Info

	if task.TotalWork > 0 {
		task.WorkStatus = make([]string, task.TotalWork)
	}
	task.RemainWork = task.TotalWork

	for _, io := range task.Inputs {
		if io.Node == "" {
			io.Node = "-"
		}
	}
	for _, io := range task.Outputs {
		if io.Node == "" {
			io.Node = "-"
		}
	}

	task.setTokenForIO()
	task.State = TASK_STAT_INIT
	return
}

func (task *Task) UpdateState(newState string) string {
	task.State = newState
	return task.State
}

//get part size based on partition/index info
//if fail to get index info, task.TotalWork fall back to 1 and return nil
func (task *Task) InitPartIndex() (err error) {
	if task.TotalWork == 1 {
		return
	}
	var input_io *IO
	if task.Partition == nil {
		if len(task.Inputs) == 1 {
			for filename, io := range task.Inputs {
				input_io = io
				task.Partition = new(PartInfo)
				task.Partition.Input = filename
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
		if _, ok := task.Inputs[task.Partition.Input]; !ok {
			task.setTotalWork(1)
			logger.Error("warning: invalid partition info, taskid=" + task.Id)
			return
		}
		input_io = task.Inputs[task.Partition.Input]
	}

	var totalunits int

	idxinfo, err := input_io.GetIndexInfo()
	if err != nil {
		task.setTotalWork(1)
		logger.Error("warning: invalid file info, taskid=" + task.Id)
		return nil
	}

	idxtype := conf.DEFAULT_INDEX
	if _, ok := idxinfo[idxtype]; !ok { //if index not available, create index
		if err := ShockPutIndex(input_io.Host, input_io.Node, idxtype, task.Info.DataToken); err != nil {
			task.setTotalWork(1)
			logger.Error("warning: fail to create index on shock for taskid=" + task.Id)
			return nil
		}
		totalunits, err = input_io.TotalUnits(idxtype) //get index info again
		if err != nil {
			task.setTotalWork(1)
			logger.Error("warning: fail to get index units, taskid=" + task.Id + ":" + err.Error())
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

func (task *Task) ParseWorkunit() (wus []*Workunit, err error) {
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

func (task *Task) Skippable() bool {
	// For a task to be skippable, it should meet
	// the following requirements (this may change
	// in the future):
	// 1.- It should have exactly one input file
	// and one output file (This way, we can connect tasks
	// Ti-1 and Ti+1 transparently)
	// 2.- It should be a simple pipeline task. That is,
	// it should just have at most one "parent" Ti-1 ---> Ti
	return (len(task.Inputs) == 1) &&
		(len(task.Outputs) == 1) &&
		(len(task.DependsOn) <= 1)
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
