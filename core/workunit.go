package core

import (
	"fmt"
)

type Workunit struct {
	Id      string   `bson:"wuid" json:"wuid"`
	Info    *Info    `bson:"info" json:"info"`
	Inputs  IOmap    `bson:"inputs" json:"inputs"`
	Outputs IOmap    `bson:"outputs" json:"outputs"`
	Cmd     *Command `bson:"cmd" json:"cmd"`
	Rank    int      `bson:"rank" json:"rank"`
}

func NewWorkunit(task *Task, rank int) *Workunit {
	return &Workunit{
		Id:      fmt.Sprintf("%s_%d", task.Id, rank),
		Info:    task.Info,
		Inputs:  NewIOmap(),
		Outputs: NewIOmap(),
		Cmd:     task.Cmd,
		Rank:    rank,
	}
}
