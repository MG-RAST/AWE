package core

import (
	"fmt"
	. "github.com/MG-RAST/AWE/logger"
	"os/exec"
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
	WorkStatus []string  `bson:"workstatus" json:"workstatus"`
	State      string    `bson:"state" json:"state"`
}

type IOmap map[string]*IO

type IO struct {
	Name   string `bson:"name" json:"name"`
	Host   string `bson:"host" json:"host"`
	Node   string `bson:"node" json:"node"`
	MD5    string `bson:"md5" json:"md5"`
	Cache  bool   `bson:"cache" json:"cache"`
	Origin string `bson:"origin" json:"origin"`
	Path   string `bson:"path" json:"-"`
}

type PartInfo struct {
	Input      string `bson:"input" json:"input"`
	Ouput      string `bson:"output" json:"output"`
	Index      string `bson:"index" json:"index"`
	TotalIndex int    `bson:"totalindex" json:"totalindex"`
	Options    string `bson:"options" json:"options"`
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

func NewIOmap() IOmap {
	return IOmap{}
}

func (i IOmap) Add(name string, host string, node string, params string, md5 string, cache bool) {
	i[name] = &IO{Name: name, Host: host, Node: node, MD5: md5, Cache: cache}
	return
}

func (i IOmap) Has(name string) bool {
	if _, has := i[name]; has {
		return true
	}
	return false
}

func (i IOmap) Find(name string) *IO {
	if val, has := i[name]; has {
		return val
	}
	return nil
}

func NewIO() *IO {
	return &IO{}
}

func (io *IO) Url() string {
	if io.Host != "" && io.Node != "" {
		return fmt.Sprintf("%s/node/%s?download", io.Host, io.Node)
	}
	return ""
}

func (io *IO) TotalUnits(indextype string) (count int, err error) {
	count, err = GetIndexUnits(indextype, io)
	return
}

func (task *Task) UpdateState(newState string) string {
	task.State = newState
	return task.State
}

// fill some info (lacked in input json) for a task 
func (task *Task) InitTask(job *Job) (err error) {
	task.Id = fmt.Sprintf("%s_%s", job.Id, task.Id)
	task.Info = job.Info
	task.State = TASK_STAT_INIT
	task.WorkStatus = make([]string, task.TotalWork)
	task.RemainWork = task.TotalWork

	for j := 0; j < len(task.DependsOn); j++ {
		depend := task.DependsOn[j]
		task.DependsOn[j] = fmt.Sprintf("%s_%s", job.Id, depend)
	}
	return
}

//get part size based on partition/index info
//if fail to get index info, task.TotalWork fall back to 1
func (task *Task) InitPartIndex() (err error) {
	if task.TotalWork == 1 {
		return
	}
	if task.TotalWork > 1 {
		if task.Partition == nil {
			task.setTotalWork(1)
			Log.Error("warning: lacking partition info while totalwork > 1, taskid=" + task.Id)
			return
		}
		var totalunits int
		if io, ok := task.Inputs[task.Partition.Input]; ok {
			if task.Partition.Index == "record" {
				totalunits, err = io.TotalUnits("record")
				if err != nil || totalunits == 0 { //if index not available, create index
					createIndex(io.Host, io.Node, "record")
				}
				totalunits, err = io.TotalUnits("record") //get index info again
				if err != nil {
					task.setTotalWork(1)
					Log.Error("warning: fail to get index units, taskid=" + task.Id + ":" + err.Error())
					return
				}
				if totalunits < task.TotalWork {
					task.TotalWork = totalunits
				}
				task.Partition.TotalIndex = totalunits
			}
		} else {
			Log.Error("warning: invaid partition info, taskid=" + task.Id)
			task.setTotalWork(1)
		}
	}
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
