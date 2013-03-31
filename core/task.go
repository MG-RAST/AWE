package core

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/conf"
	. "github.com/MG-RAST/AWE/logger"
	"os/exec"
	"strconv"
)

const (
	TASK_STAT_INIT      = "init"
	TASK_STAT_QUEUED    = "queued"
	TASK_STAT_PENDING   = "pending"
	TASK_STAT_SUSPEND   = "suspend"
	TASK_STAT_COMPLETED = "completed"
)

type Task struct {
	Id         string    `bson:"taskid" json:"taskid"`
	Info       *Info     `bson:"info" json:"-"`
	Inputs     IOmap     `bson:"inputs" json:"inputs"`
	Outputs    IOmap     `bson:"outputs" json:"outputs"`
	Cmd        *Command  `bson:"cmd" json:"cmd"`
	Partition  *PartInfo `bson:"partinfo" json:"partinfo"`
	DependsOn  []string  `bson:"dependsOn" json:"dependsOn"`
	TotalWork  int       `bson:"totalwork" json:"totalwork"`
	RemainWork int       `bson:"remainwork" json:"remainwork"`
	WorkStatus []string  `bson:"workstatus" json:"-"`
	State      string    `bson:"state" json:"state"`
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
	}
}

// fill some info (lacked in input json) for a task 
func (task *Task) InitTask(job *Job, rank int) (err error) {
	if idInt, err := strconv.Atoi(task.Id); err == nil {
		if rank != idInt {
			return errors.New(fmt.Sprintf("invalid job script: task id doen't match stage %d vs %d", rank, idInt))
		}
	} else {
		return errors.New("invalid job script: task id (stage) can't be converted to an integer: " + task.Id)
	}
	task.Id = fmt.Sprintf("%s_%s", job.Id, task.Id)
	task.Info = job.Info
	task.State = TASK_STAT_INIT
	if task.TotalWork > 0 {
		task.WorkStatus = make([]string, task.TotalWork)
	}
	task.RemainWork = task.TotalWork
	for j := 0; j < len(task.DependsOn); j++ {
		depend := task.DependsOn[j]
		task.DependsOn[j] = fmt.Sprintf("%s_%s", job.Id, depend)
	}
	return
}

func (task *Task) UpdateState(newState string) string {
	task.State = newState
	return task.State
}

//get part size based on partition/index info
//if fail to get index info, task.TotalWork fall back to 1 and return nil
func (task *Task) InitPartIndex() (err error) {
	if task.Partition == nil {
		if task.TotalWork > 1 {
			Log.Error("warning: lacking partition info while totalwork > 1, taskid=" + task.Id)
		}
		task.setTotalWork(1)
		return
	}
	if task.Partition.MaxPartSizeMB == 0 && task.TotalWork <= 1 {
		task.setTotalWork(1)
		return
	}
	var totalunits int
	if _, ok := task.Inputs[task.Partition.Input]; !ok {
		task.setTotalWork(1)
		Log.Error("warning: invalid partition info, taskid=" + task.Id)
		return
	}
	io := task.Inputs[task.Partition.Input]

	idxinfo, err := io.GetIndexInfo()
	if err != nil {
		task.setTotalWork(1)
		Log.Error("warning: invalid file info, taskid=" + task.Id)
		return nil
	}

	idxtype := conf.DEFAULT_INDEX
	if _, ok := idxinfo[idxtype]; !ok { //if index not available, create index
		if err := createIndex(io.Host, io.Node, idxtype); err != nil {
			task.setTotalWork(1)
			Log.Error("warning: fail to create index on shock for taskid=" + task.Id)
			return nil
		}
		totalunits, err = io.TotalUnits(idxtype) //get index info again
		if err != nil {
			task.setTotalWork(1)
			Log.Error("warning: fail to get index units, taskid=" + task.Id + ":" + err.Error())
			return nil
		}
	} else { //index existing, use it directly
		totalunits = idxinfo[idxtype].TotalUnits
	}

	//adjust total work based on needs	
	if task.Partition.MaxPartSizeMB > 0 { // fixed max part size
		//this implementation for chunkrecord indexer only
		chunkmb := int(conf.DEFAULT_CHUNK_SIZE / 1048576)
		if totalunits*chunkmb%task.Partition.MaxPartSizeMB == 0 {
			task.setTotalWork(totalunits * chunkmb / task.Partition.MaxPartSizeMB)
		} else {
			totalwork := totalunits*chunkmb/task.Partition.MaxPartSizeMB + 1
			task.setTotalWork(totalwork)
		}
	} else {
		if totalunits < task.TotalWork {
			task.setTotalWork(totalunits)
		}
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

//creat index
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
