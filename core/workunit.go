package core

import (
	"fmt"
	"github.com/MG-RAST/AWE/conf"
	"os"
)

const (
	WORK_STAT_QUEUED   = "queued"
	WORK_STAT_CHECKOUT = "checkout"
	WORK_STAT_SUSPEND  = "suspend"
)

type Workunit struct {
	Id        string    `bson:"wuid" json:"wuid"`
	Info      *Info     `bson:"info" json:"info"`
	Inputs    IOmap     `bson:"inputs" json:"inputs"`
	Outputs   IOmap     `bson:"outputs" json:"outputs"`
	Cmd       *Command  `bson:"cmd" json:"cmd"`
	Rank      int       `bson:"rank" json:"rank"`
	TotalWork int       `bson:"totalwork" json:"totalwork"`
	Partition *PartInfo `bson:"part" json:"part"`
	State     string    `bson:"state" json:"state"`
}

type datapart struct {
	Index string `bson:"index" json:"index"`
	Range string `bson:"range" json:"range"`
}

func NewWorkunit(task *Task, rank int) *Workunit {
	return &Workunit{
		Id:        fmt.Sprintf("%s_%d", task.Id, rank),
		Info:      task.Info,
		Inputs:    task.Inputs,
		Outputs:   task.Outputs,
		Cmd:       task.Cmd,
		Rank:      rank,
		TotalWork: task.TotalWork, //keep this info in workunit for load balancing
		Partition: task.Partition,
		State:     WORK_STAT_QUEUED,
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

//calculate the range of data part
//algorithm: try to evenly distribute indexed parts to workunits
//e.g. totalWork=4, totalParts=10, then each workunits have parts 3,3,2,2 
func (work *Workunit) Part() (part string) {
	if work.Rank == 0 {
		return ""
	}
	if work.Partition.Index == "record" {
		partsize := work.Partition.TotalIndex / work.TotalWork //floor
		remainder := work.Partition.TotalIndex % work.TotalWork
		var start, end int
		if work.Rank <= remainder {
			start = (partsize+1)*(work.Rank-1) + 1
			end = start + partsize
		} else {
			start = (partsize+1)*remainder + partsize*(work.Rank-remainder-1) + 1
			end = start + partsize - 1
		}
		if start == end {
			part = fmt.Sprint("%d", start)
		} else {
			part = fmt.Sprintf("%d-%d", start, end)
		}
	}
	return
}
