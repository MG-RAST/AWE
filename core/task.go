package core

import ()

type TaskList []Task

type Task struct {
	Info      *Info      `bson:"info" json:"info"`
	Inputs    IOmap      `bson:"inputs" json:"inputs"`
	Outputs   IOmap      `bson:"outputs" json:"outputs"`
	Cmd       *Command   `bson:"cmd" json:"cmd"`
	Partition *Partition `bson:"partion" json:"partion"`
	DependsOn []string   `bson:"dependsOn" json:"dependsOn"`
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

//func NewTaskList() []TaskList {
//	return []Task{}
//}

func NewTask() *Task {
	return &Task{
		Info:      NewInfo(),
		Inputs:    NewIOmap(),
		Outputs:   NewIOmap(),
		Cmd:       &Command{},
		Partition: nil,
		DependsOn: []string{},
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

//func (t *Task) 
