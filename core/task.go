package core

import (
	"fmt"
)

type Task struct {
	Id         string     `bson:"taskid" json:"taskid"`
	Info       *Info      `bson:"info" json:"info"`
	Inputs     IOmap      `bson:"inputs" json:"inputs"`
	Outputs    IOmap      `bson:"outputs" json:"outputs"`
	Cmd        *Command   `bson:"cmd" json:"cmd"`
	Partition  *Partition `bson:"partition" json:"partition"`
	DependsOn  []string   `bson:"dependsOn" json:"dependsOn"`
	TotalWork  int        `bson:"totalwork" json:"totalwork"`
	RemainWork int        `bson:"remainWork" json:"remainWork"`
	State      string     `bson:"state" json:"state"`
}

type IOmap map[string]*IO

type IO struct {
	Name  string `bson:"name" json:"name"`
	Url   string `bson:"url" json:"url"`
	MD5   string `bson:"md5" json:"md5"`
	Cache bool   `bson:"cache" json:"cache"`
}

type Partition struct {
	Input   string `bson:"input" json:"input"`
	Index   string `bson:"index" json:"index"`
	Options string `bson:"options" json:"options"`
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
		State:      "init",
	}
}

func NewIOmap() IOmap {
	return IOmap{}
}

func (i IOmap) Add(name string, url string, md5 string, cache bool) {
	i[name] = &IO{Name: name, Url: url, MD5: md5, Cache: cache}
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

//---Field update functions
func (task *Task) UpdateState(newState string) string {
	task.State = newState
	return task.State
}

func (task *Task) ParseWorkunit() (wus []*Workunit, err error) {
	for i := 0; i < task.TotalWork; i++ {
		workunit := NewWorkunit(task, i)
		wus = append(wus, workunit)
	}
	return
}
