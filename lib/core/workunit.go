package core

import (
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"os"
	"time"
)

const (
	WORK_STAT_QUEUED      = "queued"
	WORK_STAT_CHECKOUT    = "checkout"
	WORK_STAT_SUSPEND     = "suspend"
	WORK_STAT_DONE        = "done"
	WORK_STAT_FAIL        = "fail"
	WORK_STAT_PREPARED    = "prepared"
	WORK_STAT_COMPUTED    = "computed"
	WORK_STAT_PROXYQUEUED = "proxyqueued"
)

type Workunit struct {
	Id           string    `bson:"wuid" json:"wuid"`
	Info         *Info     `bson:"info" json:"info"`
	Inputs       IOmap     `bson:"inputs" json:"inputs"`
	Outputs      IOmap     `bson:"outputs" json:"outputs"`
	Predata      IOmap     `bson:"predata" json:"predata"`
	Cmd          *Command  `bson:"cmd" json:"cmd"`
	Rank         int       `bson:"rank" json:"rank"`
	TotalWork    int       `bson:"totalwork" json:"totalwork"`
	Partition    *PartInfo `bson:"part" json:"part"`
	State        string    `bson:"state" json:"state"`
	Failed       int       `bson:"failed" json:"failed"`
	CheckoutTime time.Time `bson:"checkout_time" json:"checkout_time"`
	Client       string    `bson:"client" json:"client"`
}

func NewWorkunit(task *Task, rank int) *Workunit {
	return &Workunit{
		Id:        fmt.Sprintf("%s_%d", task.Id, rank),
		Info:      task.Info,
		Inputs:    task.Inputs,
		Outputs:   task.Outputs,
		Predata:   task.Predata,
		Cmd:       task.Cmd,
		Rank:      rank,
		TotalWork: task.TotalWork, //keep this info in workunit for load balancing
		Partition: task.Partition,
		State:     WORK_STAT_QUEUED,
		Failed:    0,
	}
}

func (work *Workunit) Mkdir() (err error) {
	err = os.MkdirAll(work.Path(), 0777)
	if err != nil {
		return
	}
	return
}

func (work *Workunit) RemoveDir() (err error) {
	err = os.RemoveAll(work.Path())
	if err != nil {
		return
	}
	return
}

func (work *Workunit) Path() string {
	id := work.Id
	return fmt.Sprintf("%s/%s/%s/%s/%s", conf.WORK_PATH, id[0:2], id[2:4], id[4:6], id)
}

func (work *Workunit) CDworkpath() (err error) {
	return os.Chdir(work.Path())
}

func (work *Workunit) IndexType() (indextype string) {
	return work.Partition.Index
}

//calculate the range of data part
//algorithm: try to evenly distribute indexed parts to workunits
//e.g. totalWork=4, totalParts=10, then each workunits have parts 3,3,2,2
func (work *Workunit) Part() (part string) {
	if work.Rank == 0 {
		return ""
	}
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
		part = fmt.Sprintf("%d", start)
	} else {
		part = fmt.Sprintf("%d-%d", start, end)
	}
	return
}
