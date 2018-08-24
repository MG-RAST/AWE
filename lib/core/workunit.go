package core

import (
	//"bytes"
	//"encoding/json"
	"fmt"
	"reflect"

	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core/cwl"

	//"github.com/davecgh/go-spew/spew"

	//"gopkg.in/mgo.v2/bson"
	"os"
	//"path"
	//"reflect"
	//"regexp"
	"regexp/syntax"
	"strings"
	"time"
)

const (
	WORK_STAT_INIT             = "init"             // initial state
	WORK_STAT_QUEUED           = "queued"           // after requeue ; after failures below max ; on WorkQueue.Add()
	WORK_STAT_RESERVED         = "reserved"         // short lived state between queued and checkout. when a worker checks the workunit out, the state is reserved.
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
	Workunit_Unique_Identifier `bson:",inline" json:",inline" mapstructure:",squash"`
	Id                         string                 `bson:"id,omitempty" json:"id,omitempty" mapstructure:"id,omitempty"`       // global identifier: jobid_taskid_rank (for backwards coompatibility only)
	WuId                       string                 `bson:"wuid,omitempty" json:"wuid,omitempty" mapstructure:"wuid,omitempty"` // deprecated !
	Info                       *Info                  `bson:"info,omitempty" json:"info,omitempty" mapstructure:"info,omitempty"`
	Inputs                     []*IO                  `bson:"inputs,omitempty" json:"inputs,omitempty" mapstructure:"inputs,omitempty"`
	Outputs                    []*IO                  `bson:"outputs,omitempty" json:"outputs,omitempty" mapstructure:"outputs,omitempty"`
	Predata                    []*IO                  `bson:"predata,omitempty" json:"predata,omitempty" mapstructure:"predata,omitempty"`
	Cmd                        *Command               `bson:"cmd,omitempty" json:"cmd,omitempty" mapstructure:"cmd,omitempty"`
	TotalWork                  int                    `bson:"totalwork,omitempty" json:"totalwork,omitempty" mapstructure:"totalwork,omitempty"`
	Partition                  *PartInfo              `bson:"part,omitempty" json:"part,omitempty" mapstructure:"part,omitempty"`
	State                      string                 `bson:"state,omitempty" json:"state,omitempty" mapstructure:"state,omitempty"`
	Failed                     int                    `bson:"failed,omitempty" json:"failed,omitempty" mapstructure:"failed,omitempty"`
	CheckoutTime               time.Time              `bson:"checkout_time,omitempty" json:"checkout_time,omitempty" mapstructure:"checkout_time,omitempty"`
	Client                     string                 `bson:"client,omitempty" json:"client,omitempty" mapstructure:"client,omitempty"`
	ComputeTime                int                    `bson:"computetime,omitempty" json:"computetime,omitempty" mapstructure:"computetime,omitempty"`
	ExitStatus                 int                    `bson:"exitstatus,omitempty" json:"exitstatus,omitempty" mapstructure:"exitstatus,omitempty"` // Linux Exit Status Code (0 is success)
	Notes                      []string               `bson:"notes,omitempty" json:"notes,omitempty" mapstructure:"notes,omitempty"`
	UserAttr                   map[string]interface{} `bson:"userattr,omitempty" json:"userattr,omitempty" mapstructure:"userattr,omitempty"`
	ShockHost                  string                 `bson:"shockhost,omitempty" json:"shockhost,omitempty" mapstructure:"shockhost,omitempty"` // specifies default Shock host for outputs
	CWL_workunit               *CWL_workunit          `bson:"cwl,omitempty" json:"cwl,omitempty" mapstructure:"cwl,omitempty"`
	WorkPath                   string                 // this is the working directory. If empty, it will be computed.
	WorkPerf                   *WorkPerf
}

func NewWorkunit(qm *ServerMgr, task *Task, rank int, job *Job) (workunit *Workunit, err error) {
	task_id := task.Task_Unique_Identifier
	workunit = &Workunit{
		Workunit_Unique_Identifier: New_Workunit_Unique_Identifier(task_id, rank),
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
	var work_str string
	work_str, err = workunit.String()
	if err != nil {
		err = fmt.Errorf("(NewWorkunit) workunit.String() returned: %s", err.Error())
		return
	}

	workunit.Id = work_str
	workunit.WuId = work_str

	if task.WorkflowStep != nil {

		workflow_step := task.WorkflowStep

		workunit.CWL_workunit = &CWL_workunit{}

		workunit.ShockHost = job.ShockHost

		// ****** get CommandLineTool (or whatever can be executed)
		p := workflow_step.Run

		if p == nil {
			err = fmt.Errorf("(NewWorkunit) process is nil !!?")
			return
		}

		var schemata []cwl.CWLType_Type
		schemata, err = job.CWL_collection.GetSchemata()
		if err != nil {
			err = fmt.Errorf("(updateJobTask) job.CWL_collection.GetSchemata() returned: %s", err.Error())
			return
		}

		//var process_name string
		var clt *cwl.CommandLineTool
		//var a_workflow *cwl.Workflow
		var process interface{}
		process, _, err = cwl.GetProcess(p, job.CWL_collection, job.CwlVersion, schemata) // TODO add new schemata
		if err != nil {
			err = fmt.Errorf("(NewWorkunit) GetProcess returned: %s", err.Error())
			return
		}

		if process == nil {
			err = fmt.Errorf("(NewWorkunit) process == nil")
			return
		}

		//var requirements *[]cwl.Requirement

		switch process.(type) {
		case *cwl.CommandLineTool:
			clt, _ = process.(*cwl.CommandLineTool)
			if clt.CwlVersion == "" {
				clt.CwlVersion = job.CwlVersion
			}
			if clt.CwlVersion == "" {
				err = fmt.Errorf("(NewWorkunit) CommandLineTool misses CwlVersion")
				return
			}
			//requirements = clt.Requirements
		case *cwl.ExpressionTool:
			var et *cwl.ExpressionTool
			et, _ = process.(*cwl.ExpressionTool)
			if et.CwlVersion == "" {
				et.CwlVersion = job.CwlVersion
			}
			if et.CwlVersion == "" {
				err = fmt.Errorf("(NewWorkunit) ExpressionTool misses CwlVersion")
				return
			}
			//requirements = et.Requirements
		default:
			err = fmt.Errorf("(NewWorkunit) Tool %s not supported", reflect.TypeOf(process))
			return
		}

		var shock_requirement *cwl.ShockRequirement
		shock_requirement = job.CWL_ShockRequirement
		if shock_requirement == nil {
			err = fmt.Errorf("(NewWorkunit) shock_requirement == nil")
			return
		}
		//shock_requirement, err = cwl.GetShockRequirement(requirements)
		//if err != nil {
		//fmt.Println("process:")
		//spew.Dump(process)
		//	err = fmt.Errorf("(NewWorkunit) ShockRequirement not found , err: %s", err.Error())
		//	return
		//}

		if shock_requirement.Shock_api_url == "" {
			err = fmt.Errorf("(NewWorkunit) Shock_api_url in ShockRequirement is empty")
			return
		}

		workunit.ShockHost = shock_requirement.Shock_api_url

		workunit.CWL_workunit.Tool = process

		//}

		//if use_workflow {
		//	wfl.CwlVersion = job.CwlVersion
		//}

		// ****** get inputs
		job_input_map := *job.CWL_collection.Job_input_map
		if job_input_map == nil {
			err = fmt.Errorf("(NewWorkunit) job.CWL_collection.Job_input_map is empty")
			return
		}
		//job_input_map := *job.CWL_collection.Job_input_map

		//job_input := *job.CWL_collection.Job_input

		var workflow_instance *WorkflowInstance
		workflow_instance, err = job.GetWorkflowInstance(task.Parent, true)
		if err != nil {
			err = fmt.Errorf("(NewWorkunit) GetWorkflowInstance returned %s", err.Error())
			return
		}

		workflow_input_map := workflow_instance.Inputs.GetMap()

		var workunit_input_map map[string]cwl.CWLType
		workunit_input_map, err = qm.GetStepInputObjects(job, task_id, workflow_input_map, workflow_step)
		if err != nil {
			err = fmt.Errorf("(NewWorkunit) GetStepInputObjects returned: %s", err.Error())
			return
		}

		//fmt.Println("workunit_input_map after second round:\n")
		//spew.Dump(workunit_input_map)

		job_input := cwl.Job_document{}

		for elem_id, elem := range workunit_input_map {
			named_type := cwl.NewNamedCWLType(elem_id, elem)
			job_input = append(job_input, named_type)
		}

		workunit.CWL_workunit.Job_input = &job_input

		//spew.Dump(job_input)

		workunit.CWL_workunit.OutputsExpected = &workflow_step.Out

		err = workunit.Evaluate(workunit_input_map)
		if err != nil {
			err = fmt.Errorf("(NewWorkunit) workunit.Evaluate returned: %s", err.Error())
			return
		}
		//spew.Dump(workflow_step.Out)
		//panic("done")

	}
	//panic("done")
	//spew.Dump(workunit.Cmd)
	//panic("done")

	return
}

func (w *Workunit) Evaluate(inputs interface{}) (err error) {

	if w.CWL_workunit != nil {
		process := w.CWL_workunit.Tool
		switch process.(type) {
		case *cwl.CommandLineTool:
			clt := process.(*cwl.CommandLineTool)

			err = clt.Evaluate(inputs)
			if err != nil {
				err = fmt.Errorf("(Workunit/Evaluate) CommandLineTool.Evaluate returned: %s", err.Error())
				return
			}
		case *cwl.ExpressionTool:
			et := process.(*cwl.ExpressionTool)

			err = et.Evaluate(inputs)
			if err != nil {
				err = fmt.Errorf("(Workunit/Evaluate) ExpressionTool.Evaluate returned: %s", err.Error())
				return
			}

		case *cwl.Workflow:
			wf := process.(*cwl.Workflow)
			err = wf.Evaluate(inputs)
			if err != nil {
				err = fmt.Errorf("(Workunit/Evaluate) Workflow.Evaluate returned: %s", err.Error())
				return
			}

		default:
			err = fmt.Errorf("(Workunit/Evaluate) Process type not supported %s", reflect.TypeOf(process))
			return
		}
	}
	return
}

func (w *Workunit) GetId() (id Workunit_Unique_Identifier) {
	id = w.Workunit_Unique_Identifier
	return
}

func (work *Workunit) Mkdir() (err error) {
	// delete workdir just in case it exists; will not work if awe-worker is not in docker container AND tasks are in container
	work_path, err := work.Path()
	if err != nil {
		return
	}
	os.RemoveAll(work_path)
	err = os.MkdirAll(work_path, 0777)
	if err != nil {
		return
	}
	return
}

func (work *Workunit) RemoveDir() (err error) {
	work_path, err := work.Path()
	if err != nil {
		return
	}
	err = os.RemoveAll(work_path)
	if err != nil {
		return
	}
	return
}

func (work *Workunit) SetState(new_state string, reason string) (err error) {

	if new_state == WORK_STAT_SUSPEND && reason == "" {
		err = fmt.Errorf("To suspend you need to provide a reason")
		return
	}

	work.State = new_state
	if new_state != WORK_STAT_CHECKOUT {
		work.Client = ""
	}

	if reason != "" {
		if len(work.Notes) == 0 {
			work.Notes = append(work.Notes, reason)
		}
	}

	return
}

func (work *Workunit) Path() (path string, err error) {
	if work.WorkPath == "" {
		id := work.Workunit_Unique_Identifier.JobId

		if id == "" {
			err = fmt.Errorf("(Workunit/Path) JobId is missing")
			return
		}
		task_name := work.Workunit_Unique_Identifier.Parent
		if task_name != "" {
			task_name += "-"
		}
		task_name += work.Workunit_Unique_Identifier.TaskName
		// convert name to make it filesystem compatible
		task_name = strings.Map(
			func(r rune) rune {
				if syntax.IsWordChar(r) || r == '-' { // word char: [0-9A-Za-z] and '-'
					return r
				}
				return '_'
			},
			task_name)

		work.WorkPath = fmt.Sprintf("%s/%s/%s/%s/%s_%s_%d", conf.WORK_PATH, id[0:2], id[2:4], id[4:6], id, task_name, work.Workunit_Unique_Identifier.Rank)
	}
	path = work.WorkPath
	return
}

func (work *Workunit) CDworkpath() (err error) {
	work_path, err := work.Path()
	if err != nil {
		return
	}
	return os.Chdir(work_path)
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
