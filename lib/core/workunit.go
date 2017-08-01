package core

import (
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	WORK_STAT_INIT             = "init"             // initial state
	WORK_STAT_QUEUED           = "queued"           // after requeue ; after failures below max ; on WorkQueue.Add()
	WORK_STAT_RESERVED         = "reserved"         // short lived state between queued and checkout
	WORK_STAT_CHECKOUT         = "checkout"         // normal work checkout ; client registers that already has a workunit (e.g. after reboot of server)
	WORK_STAT_SUSPEND          = "suspend"          // on MAX_FAILURE ; on SuspendJob
	WORK_STAT_FAILED_PERMANENT = "failed-permanent" // app had exit code 42
	WORK_STAT_DONE             = "done"             // client only: done
	WORK_STAT_ERROR            = "fail"             // client only: workunit computation or IO error (variable was renamed to ERROR but not the string fail, to maintain backwards compability)
	WORK_STAT_PREPARED         = "prepared"         // client only: after argument parsing
	WORK_STAT_COMPUTED         = "computed"         // client only: after computation is done, before upload
	WORK_STAT_DISCARDED        = "discarded"        // client only: job / task suspended or server UUID changes
	WORK_STAT_PROXYQUEUED      = "proxyqueued"      // proxy only
)

type Workunit struct {
	Workunit_Unique_Identifier `bson:",inline"`
	Id                         string `bson:"id" json:"id"` // global identifier: jobid_taskid_rank

	Info         *Info             `bson:"info" json:"info"`
	Inputs       []*IO             `bson:"inputs" json:"inputs"`
	Outputs      []*IO             `bson:"outputs" json:"outputs"`
	Predata      []*IO             `bson:"predata" json:"predata"`
	Cmd          *Command          `bson:"cmd" json:"cmd"`
	TotalWork    int               `bson:"totalwork" json:"totalwork"`
	Partition    *PartInfo         `bson:"part" json:"part"`
	State        string            `bson:"state" json:"state"`
	Failed       int               `bson:"failed" json:"failed"`
	CheckoutTime time.Time         `bson:"checkout_time" json:"checkout_time"`
	Client       string            `bson:"client" json:"client"`
	ComputeTime  int               `bson:"computetime" json:"computetime"`
	ExitStatus   int               `bson:"exitstatus" json:"exitstatus"` // Linux Exit Status Code (0 is success)
	Notes        []string          `bson:"notes" json:"notes"`
	UserAttr     map[string]string `bson:"userattr" json:"userattr"`
	WorkPath     string            // this is the working directory. If empty, it will be computed.
	WorkPerf     *WorkPerf
	CWL          *CWL_workunit
}

type Workunit_Unique_Identifier struct {
	Rank   int    `bson:"rank" json:"rank"` // this is the local identifier
	TaskId string `bson:"taskid" json:"taskid"`
	JobId  string `bson:"jobid" json:"jobid"`
}

func (w Workunit_Unique_Identifier) String() string {
	return fmt.Sprintf("%s_%s_%d", w.JobId, w.TaskId, w.Rank)
}

func (w Workunit_Unique_Identifier) GetTask() Task_Unique_Identifier {
	return Task_Unique_Identifier{JobId: w.JobId, Id: w.TaskId}
}

func New_Workunit_Unique_Identifier(old_style_id string) (w Workunit_Unique_Identifier, err error) {

	array := strings.Split(old_style_id, "_")

	if len(array) != 3 {
		err = fmt.Errorf("Cannot parse workunit identifier: %s", old_style_id)
		return
	}

	rank, err := strconv.Atoi(array[2])
	if err != nil {
		return
	}

	w = Workunit_Unique_Identifier{JobId: array[0], TaskId: array[1], Rank: rank}

	return
}

type CWL_workunit struct {
	Job_input          *cwl.Job_document
	Job_input_filename string
	CWL_tool           *cwl.CommandLineTool
	CWL_tool_filename  string
	Tool_results       *cwl.Job_document
	OutputsExpected    *cwl.WorkflowStepOutput // this is the subset of outputs that are needed by the workflow
}

func NewCWL_workunit() *CWL_workunit {
	return &CWL_workunit{
		Job_input:       nil,
		CWL_tool:        nil,
		Tool_results:    nil,
		OutputsExpected: nil,
	}

}

type WorkLog struct {
	Id   string            `bson:"wuid" json:"wuid"`
	Rank int               `bson:"rank" json:"rank"`
	Logs map[string]string `bson:"logs" json:"logs"`
}

func NewWorkLog(id Workunit_Unique_Identifier) (wlog *WorkLog) {
	work_id := fmt.Sprintf("%s_%d", id.TaskId, id.Rank)
	wlog = new(WorkLog)
	wlog.Id = work_id
	wlog.Rank = id.Rank
	wlog.Logs = map[string]string{}
	for _, log := range conf.WORKUNIT_LOGS {

		wlog.Logs[log], _ = QMgr.GetReportMsg(id, log)
	}
	return
}

// create workunit slice type to use for sorting

type WorkunitsSortby struct {
	Order     string
	Direction string
	Workunits []*Workunit
}

func (w *Workunit) GetUniqueIdentifier() (id Workunit_Unique_Identifier) {
	id = Workunit_Unique_Identifier{Rank: w.Rank, TaskId: w.TaskId, JobId: w.JobId}
	return
}

func (w WorkunitsSortby) Len() int {
	return len(w.Workunits)
}

func (w WorkunitsSortby) Swap(i, j int) {
	w.Workunits[i], w.Workunits[j] = w.Workunits[j], w.Workunits[i]
}

func (w WorkunitsSortby) Less(i, j int) bool {
	// default is ascending
	if w.Direction == "desc" {
		i, j = j, i
	}
	switch w.Order {
	// default is info.submittime
	default:
		return w.Workunits[i].Info.SubmitTime.Before(w.Workunits[j].Info.SubmitTime)
	case "wuid":
		return w.Workunits[i].Id < w.Workunits[j].Id
	case "client":
		return w.Workunits[i].Client < w.Workunits[j].Client
	case "info.submittime":
		return w.Workunits[i].Info.SubmitTime.Before(w.Workunits[j].Info.SubmitTime)
	case "checkout_time":
		return w.Workunits[i].CheckoutTime.Before(w.Workunits[j].CheckoutTime)
	case "info.name":
		return w.Workunits[i].Info.Name < w.Workunits[j].Info.Name
	case "cmd.name":
		return w.Workunits[i].Cmd.Name < w.Workunits[j].Cmd.Name
	case "rank":
		return w.Workunits[i].Rank < w.Workunits[j].Rank
	case "totalwork":
		return w.Workunits[i].TotalWork < w.Workunits[j].TotalWork
	case "state":
		return w.Workunits[i].State < w.Workunits[j].State
	case "failed":
		return w.Workunits[i].Failed < w.Workunits[j].Failed
	case "info.priority":
		return w.Workunits[i].Info.Priority < w.Workunits[j].Info.Priority
	}
}

func NewWorkunit(task *Task, rank int) (w *Workunit) {

	w = &Workunit{
		Workunit_Unique_Identifier: Workunit_Unique_Identifier{
			Rank:   rank,
			TaskId: task.Id,
			JobId:  task.JobId,
		},
		Id:  "defined below",
		Cmd: task.Cmd,
		//App:       task.App,
		Info:    task.Info,
		Inputs:  task.Inputs,
		Outputs: task.Outputs,
		Predata: task.Predata,

		TotalWork:  task.TotalWork, //keep this info in workunit for load balancing
		Partition:  task.Partition,
		State:      WORK_STAT_INIT,
		Failed:     0,
		UserAttr:   task.UserAttr,
		ExitStatus: -1,

		//AppVariables: task.AppVariables // not needed yet
	}

	w.Id = w.String()

	return
}

func (work *Workunit) Mkdir() (err error) {
	// delete workdir just in case it exists; will not work if awe-worker is not in docker container AND tasks are in container
	os.RemoveAll(work.Path())
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

func (work *Workunit) SetState(new_state string) {
	work.State = new_state
	if new_state != WORK_STAT_CHECKOUT {
		work.Client = ""
	}
}

func (work *Workunit) Path() string {
	if work.WorkPath == "" {
		id := work.Id
		work.WorkPath = fmt.Sprintf("%s/%s/%s/%s/%s", conf.WORK_PATH, id[0:2], id[2:4], id[4:6], id)
	}
	return work.WorkPath
}

func (work *Workunit) CDworkpath() (err error) {
	return os.Chdir(work.Path())
}

func (work *Workunit) GetNotes() string {
	seen := map[string]bool{}
	uniq := []string{}
	for _, n := range work.Notes {
		if _, ok := seen[n]; !ok {
			uniq = append(uniq, n)
			seen[n] = true
		}
	}
	return strings.Join(uniq, "###")
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
