package core

import (
	"fmt"
	"github.com/MG-RAST/AWE/conf"
	"os"
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
		Inputs:  task.Inputs,
		Outputs: task.Outputs,
		Cmd:     task.Cmd,
		Rank:    rank,
	}
}

func (work *Workunit) Mkdir() (err error) {
	err = os.MkdirAll(work.Path(), 0777)
	if err != nil {
		return
	}
	return
}

func (work *Workunit) Path() string {
	return getWorkPath(work.Id)
}

func getWorkPath(id string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s", conf.WORK_PATH, id[0:2], id[2:4], id[4:6], id)
}

func (work *Workunit) CDworkpath() (err error) {
	return os.Chdir(work.Path())
}
