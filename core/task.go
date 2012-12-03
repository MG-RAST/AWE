package core

import (
	"errors"
	"fmt"
)

type Task struct {
	Id         string    `bson:"taskid" json:"taskid"`
	Info       *Info     `bson:"info" json:"info"`
	Inputs     IOmap     `bson:"inputs" json:"inputs"`
	Outputs    IOmap     `bson:"outputs" json:"outputs"`
	Cmd        *Command  `bson:"cmd" json:"cmd"`
	Partition  *PartInfo `bson:"partinfo" json:"partinfo"`
	DependsOn  []string  `bson:"dependsOn" json:"dependsOn"`
	TotalWork  int       `bson:"totalwork" json:"totalwork"`
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
	Units  int    `bson:"units" json:"units"`
	Size   int64  `bson:"size"  json:"size"`
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
		WorkStatus: []string{},
		State:      "init",
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

func (io IO) Url() string {
	if io.Host != "" && io.Node != "" {
		return fmt.Sprintf("%s/node/%s?download", io.Host, io.Node)
	}
	return ""
}

//to-do: get io units from Shock instead of depending on job script
func (io IO) TotalUnits() (num int) {
	return io.Units
}

func (task *Task) UpdateState(newState string) string {
	task.State = newState
	return task.State
}

// fill some info (lacked in input json) for a task 
func (task *Task) InitTask(job *Job) (err error) {
	task.Id = fmt.Sprintf("%s_%s", job.Id, task.Id)
	task.Info = job.Info
	task.State = "init"
	task.WorkStatus = make([]string, task.TotalWork)

	for j := 0; j < len(task.DependsOn); j++ {
		depend := task.DependsOn[j]
		task.DependsOn[j] = fmt.Sprintf("%s_%s", job.Id, depend)
	}

	if err = task.InitPartition(); err != nil {
		return err
	}
	return
}

// calculate part size based on partition info
func (task *Task) InitPartition() (err error) {
	if task.TotalWork == 1 {
		return
	}
	if task.TotalWork > 1 {
		if task.Partition == nil {
			return errors.New(fmt.Sprintf("missing Partition Info, taskid=%s", task.Id))
		}
		if io, ok := task.Inputs[task.Partition.Input]; ok {
			if task.Partition.Index == "record" {
				totalunits := io.TotalUnits()
				if totalunits < task.TotalWork {
					task.TotalWork = totalunits
				}
				task.Partition.TotalIndex = totalunits
				return
			}
		}
		return errors.New(fmt.Sprintf("error in parsing task Partition Info, taskid=%s", task.Id))
	}
	return
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
