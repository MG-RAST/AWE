package core

import (
	//"bytes"
	//"encoding/json"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"path"
	"reflect"

	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/golib/httpclient"

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

// Workunit _
type Workunit struct {
	Workunit_Unique_Identifier `bson:",inline" json:",inline" mapstructure:",squash"`
	WorkunitState              `bson:",inline" json:",inline" mapstructure:",squash"`
	ID                         string                 `bson:"id,omitempty" json:"id,omitempty" mapstructure:"id,omitempty"`       // global identifier: jobid_taskid_rank (for backwards coompatibility only)
	WuID                       string                 `bson:"wuid,omitempty" json:"wuid,omitempty" mapstructure:"wuid,omitempty"` // deprecated !
	Info                       *Info                  `bson:"info,omitempty" json:"info,omitempty" mapstructure:"info,omitempty"` // ***
	Inputs                     []*IO                  `bson:"inputs,omitempty" json:"inputs,omitempty" mapstructure:"inputs,omitempty"`
	Outputs                    []*IO                  `bson:"outputs,omitempty" json:"outputs,omitempty" mapstructure:"outputs,omitempty"`
	Predata                    []*IO                  `bson:"predata,omitempty" json:"predata,omitempty" mapstructure:"predata,omitempty"` // ***
	Cmd                        *Command               `bson:"cmd,omitempty" json:"cmd,omitempty" mapstructure:"cmd,omitempty"`             // ***
	TotalWork                  int                    `bson:"totalwork,omitempty" json:"totalwork,omitempty" mapstructure:"totalwork,omitempty"`
	Partition                  *PartInfo              `bson:"part,omitempty" json:"part,omitempty" mapstructure:"part,omitempty"` // ***
	CheckoutTime               time.Time              `bson:"checkout_time,omitempty" json:"checkout_time,omitempty" mapstructure:"checkout_time,omitempty"`
	ComputeTime                int                    `bson:"computetime,omitempty" json:"computetime,omitempty" mapstructure:"computetime,omitempty"`
	ExitStatus                 int                    `bson:"exitstatus,omitempty" json:"exitstatus,omitempty" mapstructure:"exitstatus,omitempty"` // Linux Exit Status Code (0 is success)
	Notes                      []string               `bson:"notes,omitempty" json:"notes,omitempty" mapstructure:"notes,omitempty"`
	UserAttr                   map[string]interface{} `bson:"userattr,omitempty" json:"userattr,omitempty" mapstructure:"userattr,omitempty"`
	ShockHost                  string                 `bson:"shockhost,omitempty" json:"shockhost,omitempty" mapstructure:"shockhost,omitempty"` // specifies default Shock host for outputs
	CWLWorkunit                *CWLWorkunit           `bson:"cwl,omitempty" json:"cwl,omitempty" mapstructure:"cwl,omitempty"`
	WorkPath                   string                 // this is the working directory. If empty, it will be computed.
	WorkPerf                   *WorkPerf
	Context                    *cwl.WorkflowContext `bson:"-" json:"-" mapstructure:"-"`
}

// WorkunitState _
type WorkunitState struct {
	State  string `bson:"state,omitempty" json:"state,omitempty" mapstructure:"state,omitempty"`
	Failed int    `bson:"failed,omitempty" json:"failed,omitempty" mapstructure:"failed,omitempty"`
	Client string `bson:"client,omitempty" json:"client,omitempty" mapstructure:"client,omitempty"`
}

// NewWorkunit _
func NewWorkunit(qm *ServerMgr, task *Task, rank int, job *Job) (workunit *Workunit, err error) {

	taskID := task.Task_Unique_Identifier

	//workunit_state :=

	workunit = &Workunit{
		Workunit_Unique_Identifier: New_Workunit_Unique_Identifier(taskID, rank),
		WorkunitState: WorkunitState{
			State:  WORK_STAT_INIT,
			Failed: 0,
		},
		ID:  "defined below",
		Cmd: task.Cmd,
		//App:       task.App,
		Info:    task.Info,
		Inputs:  task.Inputs,
		Outputs: task.Outputs,
		Predata: task.Predata,

		TotalWork: task.TotalWork, //keep this info in workunit for load balancing
		Partition: task.Partition,

		UserAttr:   task.UserAttr,
		ExitStatus: -1,

		//AppVariables: task.AppVariables // not needed yet
	}
	var workStr string
	workStr, err = workunit.String()
	if err != nil {
		err = fmt.Errorf("(NewWorkunit) workunit.String() returned: %s", err.Error())
		return
	}

	logger.Debug(3, "(NewWorkunit) start: %s", workStr)

	workunit.ID = workStr
	workunit.WuID = workStr

	if task.WorkflowStep != nil {

		workflowStep := task.WorkflowStep

		taskStr, _ := task.String()

		workunit.CWLWorkunit = &CWLWorkunit{}

		workunit.ShockHost = job.ShockHost

		// ****** get CommandLineTool (or whatever can be executed)
		p := workflowStep.Run

		if p == nil {
			err = fmt.Errorf("(NewWorkunit) process is nil !!?")
			return
		}

		var clt *cwl.CommandLineTool
		var et *cwl.ExpressionTool

		//var a_workflow *cwl.Workflow
		var process interface{}
		process, _, err = cwl.GetProcess(p, job.WorkflowContext) // TODO add new schemata
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
				clt.CwlVersion = job.WorkflowContext.CwlVersion
			}
			if clt.CwlVersion == "" {
				err = fmt.Errorf("(NewWorkunit) CommandLineTool misses CwlVersion")
				return
			}

			if clt.Namespaces == nil {
				clt.Namespaces = job.WorkflowContext.Namespaces // not sure why the CommandLineTool does not have the namespaces... :-(
			}
			//requirements = clt.Requirements
		case *cwl.ExpressionTool:

			et, _ = process.(*cwl.ExpressionTool)
			if et.CwlVersion == "" {
				et.CwlVersion = job.WorkflowContext.CwlVersion
			}
			if et.CwlVersion == "" {
				err = fmt.Errorf("(NewWorkunit) ExpressionTool misses CwlVersion")
				return
			}
			if et.Namespaces == nil {
				et.Namespaces = job.WorkflowContext.Namespaces
			}
		default:
			err = fmt.Errorf("(NewWorkunit) Tool %s not supported", reflect.TypeOf(process))
			return
		}

		var shockRequirement *cwl.ShockRequirement
		shockRequirement = job.CWL_ShockRequirement
		if shockRequirement == nil {
			err = fmt.Errorf("(NewWorkunit) shockRequirement == nil")
			return
		}
		//shock_requirement, err = cwl.GetShockRequirement(requirements)
		//if err != nil {
		//fmt.Println("process:")
		//spew.Dump(process)
		//	err = fmt.Errorf("(NewWorkunit) ShockRequirement not found , err: %s", err.Error())
		//	return
		//}

		if shockRequirement.Shock_api_url == "" {
			err = fmt.Errorf("(NewWorkunit) Shock_api_url in ShockRequirement is empty")
			return
		}

		workunit.ShockHost = shockRequirement.Shock_api_url

		workunit.CWLWorkunit.Tool = process

		//}

		//if use_workflow {
		//	wfl.CwlVersion = job.CwlVersion
		//}

		context := job.WorkflowContext
		// ****** get inputs
		//job_input_map := context.Job_input_map
		//if job_input_map == nil {
		//	err = fmt.Errorf("(NewWorkunit) job.CWL_collection.Job_input_map is empty")
		//	return
		//}
		//job_input_map := *job.CWL_collection.Job_input_map

		//job_input := *job.CWL_collection.Job_input

		var workflowInstance *WorkflowInstance
		var ok bool
		workflowInstance, ok, err = job.GetWorkflowInstance(task.WorkflowInstanceID, true)
		if err != nil {
			err = fmt.Errorf("(NewWorkunit) GetWorkflowInstance returned %s", err.Error())
			return
		}
		if !ok {
			err = fmt.Errorf("(NewWorkunit) WorkflowInstance not found: \"%s\"", task.WorkflowInstanceID)
			return
		}

		workflowInputMap := workflowInstance.Inputs.GetMap()

		// remove step inputs which are not required by the tool

		toolInputsMap := make(map[string]bool)
		if clt != nil {
			for inputI := range clt.Inputs {
				commandInputParameter := &clt.Inputs[inputI]
				commandInputParameterID := commandInputParameter.ID
				commandInputParameterIDBase := path.Base(commandInputParameterID)
				toolInputsMap[commandInputParameterIDBase] = true
			}
		}
		if et != nil {
			for inputI := range et.Inputs {
				inputParameter := &et.Inputs[inputI]
				inputParameterID := inputParameter.ID
				inputParameterIDBase := path.Base(inputParameterID)
				toolInputsMap[inputParameterIDBase] = true
			}
		}

		var workflowStepInputsAll []*cwl.WorkflowStepInput

		for i := range workflowStep.In {
			stepInput := &workflowStep.In[i]

			workflowStepInputsAll = append(workflowStepInputsAll, stepInput)
		}

		//var workunitInputMapFiltered map[string]cwl.CWLType
		var workunitInputMapAll map[string]cwl.CWLType
		var reason string
		workunitInputMapAll, ok, reason, err = qm.GetStepInputObjects(job, workflowInstance, workflowInputMap, workflowStepInputsAll, context, "NewWorkunit")
		if err != nil {
			err = fmt.Errorf("(NewWorkunit) task=%s, GetStepInputObjects returned: %s", taskStr, err.Error())
			return
		}

		if !ok {
			err = fmt.Errorf("(NewWorkunit) task=%s, GetStepInputObjects not ready, reason: %s", taskStr, reason)
			return
		}

		// get Defaults from inputs such they are part of javascript evaluation

		if clt != nil {

			// check CommandLineTool for Default values
			for inputI := range clt.Inputs {
				commandInputParameter := &clt.Inputs[inputI]
				commandInputParameterID := commandInputParameter.ID
				commandInputParameterIDBase := path.Base(commandInputParameterID)

				input, hasInput := workunitInputMapAll[commandInputParameterIDBase]
				if hasInput {
					// check if input is not Null
					_, isNull := input.(*cwl.Null)
					if !isNull {
						continue // no need to add a default
					}
				}

				// check if a default exists
				if commandInputParameter.Default != nil {
					workunitInputMapAll[commandInputParameterIDBase] = commandInputParameter.Default
				}
			}
		}

		// check ExpressionTool for Default values
		if et != nil {
			for inputI := range et.Inputs {
				inputParameter := &et.Inputs[inputI]
				inputParameterID := inputParameter.ID
				inputParameterIDBase := path.Base(inputParameterID)
				_, hasInput := workunitInputMapAll[inputParameterIDBase]
				if hasInput {
					continue // no need to add a default
				}

				// check if a default exists
				if inputParameter.Default != nil {
					workunitInputMapAll[inputParameterIDBase] = inputParameter.Default
				}
			}
		}

		//	fmt.Println("workunit_input_map after second round:\n")
		//	spew.Dump(workunit_input_map)

		//fmt.Println("workflow_instance:")
		//spew.Dump(workflow_instance)
		//fmt.Println("job_input:")
		//spew.Dump(job_input)
		//fmt.Println("workflowStep.Run:")
		//spew.Dump(workflowStep.Run)
		//panic("done workflowStep.Out")
		workunit.CWLWorkunit.OutputsExpected = &workflowStep.Out

		err = workunit.Evaluate(workunitInputMapAll, context)
		if err != nil {
			err = fmt.Errorf("(NewWorkunit) workunit.Evaluate returned: %s", err.Error())
			return
		}

		jobInput := cwl.Job_document{}

		for elemID, elem := range workunitInputMapAll {
			// do not copy step input that are not needed by tool
			elemIDBase := path.Base(elemID)
			_, toolHasInput := toolInputsMap[elemIDBase]
			if !toolHasInput {
				continue
			}

			namedType := cwl.NewNamedCWLType(elemID, elem)
			jobInput = append(jobInput, namedType)
		}

		workunit.CWLWorkunit.JobInput = &jobInput

		//fmt.Println("(NewWorkunit) jobInput")
		//jobInput_bytes, _ := json.Marshal(jobInput)
		//fmt.Printf("%s\n", string(jobInput_bytes))

		// remove inputs that are not required

	}

	return
}

// Evaluate _
func (work *Workunit) Evaluate(inputs interface{}, context *cwl.WorkflowContext) (err error) {

	if work.CWLWorkunit != nil {
		process := work.CWLWorkunit.Tool
		switch process.(type) {
		case *cwl.CommandLineTool:
			clt := process.(*cwl.CommandLineTool)

			err = clt.Evaluate(inputs, context)
			if err != nil {
				err = fmt.Errorf("(Workunit/Evaluate) CommandLineTool.Evaluate returned: %s", err.Error())
				return
			}
		case *cwl.ExpressionTool:
			et := process.(*cwl.ExpressionTool)

			err = et.Evaluate(inputs, context)
			if err != nil {
				err = fmt.Errorf("(Workunit/Evaluate) ExpressionTool.Evaluate returned: %s", err.Error())
				return
			}

		case *cwl.Workflow:
			wf := process.(*cwl.Workflow)
			err = wf.Evaluate(inputs, context)
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

// GetID _
func (work *Workunit) GetID() (id Workunit_Unique_Identifier) {
	id = work.Workunit_Unique_Identifier
	return
}

// Mkdir _
func (work *Workunit) Mkdir() (err error) {
	// delete workdir just in case it exists; will not work if awe-worker is not in docker container AND tasks are in container
	workPath, err := work.Path()
	if err != nil {
		err = fmt.Errorf("(Workunit/Mkdir) work.Path() returned: %s", err.Error())
		return
	}
	err = os.RemoveAll(workPath)
	if err != nil {
		err = fmt.Errorf("(Workunit/Mkdir) os.RemoveAll returned: %s", err.Error())
		return
	}
	err = os.MkdirAll(workPath, 0777)
	if err != nil {
		err = fmt.Errorf("(Workunit/Mkdir) os.MkdirAll returned: %s", err.Error())
		return
	}
	return
}

// RemoveDir _
func (work *Workunit) RemoveDir() (err error) {
	workPath, err := work.Path()
	if err != nil {
		return
	}
	err = os.RemoveAll(workPath)
	if err != nil {
		return
	}
	return
}

// SetState _
func (work *Workunit) SetState(newState string, reason string) (err error) {

	if newState == WORK_STAT_SUSPEND && reason == "" {
		err = fmt.Errorf("To suspend you need to provide a reason")
		return
	}

	work.State = newState
	if newState != WORK_STAT_CHECKOUT {
		work.Client = ""
	}

	if reason != "" {
		if len(work.Notes) > 0 {
			work.Notes = append(work.Notes, reason)
		} else {
			work.Notes = []string{reason}
		}
	}

	return
}

// Path _
func (work *Workunit) Path() (path string, err error) {
	if work.WorkPath == "" {
		id := work.Workunit_Unique_Identifier.JobId

		if id == "" {
			err = fmt.Errorf("(Workunit/Path) JobId is missing")
			return
		}
		//task_name := work.Workunit_Unique_Identifier.Parent
		//if task_name != "" {
		//	task_name += "-"
		//}
		taskName := work.Workunit_Unique_Identifier.TaskName
		// convert name to make it filesystem compatible
		taskName = strings.Map(
			func(r rune) rune {
				if syntax.IsWordChar(r) || r == '-' { // word char: [0-9A-Za-z] and '-'
					return r
				}
				return '_'
			},
			taskName)

		work.WorkPath = fmt.Sprintf("%s/%s/%s/%s/%s_%s_%d", conf.WORK_PATH, id[0:2], id[2:4], id[4:6], id, taskName, work.Workunit_Unique_Identifier.Rank)
	}
	path = work.WorkPath
	return
}

// CDworkpath _
func (work *Workunit) CDworkpath() (err error) {
	workPath, err := work.Path()
	if err != nil {
		return
	}
	return os.Chdir(workPath)
}

// GetNotes _
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

//Part calculate the range of data part
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

// GetIDBase64 _
func (work *Workunit) GetIDBase64() (workIDB64 string, err error) {
	var workStr string
	workStr, err = work.String()
	if err != nil {
		err = fmt.Errorf("(NotifyWorkunitProcessedWithLogs) workid.String() returned: %s", err.Error())
		return
	}

	workIDB64 = "base64:" + base64.StdEncoding.EncodeToString([]byte(workStr))
	return
}

// FetchDataToken _
func (work *Workunit) FetchDataToken() (token string, err error) {

	var workIDB64 string
	workIDB64, err = work.GetIDBase64()
	if err != nil {
		err = fmt.Errorf("(FetchDataToken) work.GetIDBase64 returned: %s", err.Error())
		return
	}

	targeturl := fmt.Sprintf("%s/work/%s?datatoken&client=%s", conf.SERVER_URL, workIDB64, Self.ID)
	logger.Debug(1, "(FetchDataToken) targeturl: %s", targeturl)
	var headers httpclient.Header
	logger.Debug(3, "(FetchDataToken) len(conf.CLIENT_GROUP_TOKEN): %d ", len(conf.CLIENT_GROUP_TOKEN))
	if conf.CLIENT_GROUP_TOKEN != "" {

		headers = httpclient.Header{
			"Authorization": []string{"CG_TOKEN " + conf.CLIENT_GROUP_TOKEN},
		}
	}
	res, err := httpclient.Get(targeturl, headers, nil)
	if err != nil {
		err = fmt.Errorf("(FetchDataToken) httpclient.Get returned: %s", err.Error())
		return
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {

		var bodyBytes []byte
		bodyBytes, err = ioutil.ReadAll(res.Body)
		if err != nil {
			err = fmt.Errorf("(FetchDataToken) res.Status was %d, but could not read body: %s", res.StatusCode, err.Error())
			return
		}

		bodyString := string(bodyBytes)

		err = fmt.Errorf("(FetchDataToken) res.Status was %d, body contained: %s", res.StatusCode, bodyString)
		return
	}

	if res.Header == nil {
		logger.Debug(3, "(FetchDataToken) res.Header empty")
		return
	}

	headerArray, ok := res.Header["Datatoken"]
	if !ok {
		logger.Debug(3, "(FetchDataToken) Datatoken header not found")
		return
	}

	if len(headerArray) == 0 {
		logger.Debug(3, "(FetchDataToken) len(headerArray) == 0")
		return
	}

	token = headerArray[0]

	return
}
