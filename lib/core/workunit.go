package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core/cwl"

	"github.com/davecgh/go-spew/spew"
	"github.com/robertkrimen/otto"
	"gopkg.in/mgo.v2/bson"
	"os"
	"path"
	"reflect"
	"regexp"
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

func NewWorkunit(task *Task, rank int, job *Job) (workunit *Workunit, err error) {

	workunit = &Workunit{
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

	workunit.Id = workunit.String()
	workunit.WuId = workunit.String()

	if task.WorkflowStep != nil {

		workflow_step := task.WorkflowStep

		workunit.CWL_workunit = &CWL_workunit{}

		workunit.ShockHost = job.ShockHost

		// ****** get CommandLineTool (or whatever can be executed)
		p := workflow_step.Run

		tool_name := ""

		switch p.(type) {
		case cwl.ProcessPointer:

			pp, _ := p.(cwl.ProcessPointer)

			tool_name = pp.Value

		case bson.M: // I have no idea why we get a bson.M here

			p_bson := p.(bson.M)

			tool_name_interface, ok := p_bson["value"]
			if !ok {
				err = fmt.Errorf("(NewWorkunit) bson.M did not hold a field named value")
				return
			}

			tool_name, ok = tool_name_interface.(string)
			if !ok {
				err = fmt.Errorf("(NewWorkunit) bson.M value field is not a string")
				return
			}

		default:
			err = fmt.Errorf("(NewWorkunit) Process type %s unknown, cannot create Workunit", reflect.TypeOf(p))
			return

		}

		if tool_name == "" {
			err = fmt.Errorf("(NewWorkunit) No tool name found")
			return
		}

		if job.CWL_collection == nil {
			err = fmt.Errorf("(NewWorkunit) job.CWL_collection == nil ")
			return
		}

		clt, xerr := job.CWL_collection.GetCommandLineTool(tool_name)
		if xerr != nil {
			err = fmt.Errorf("(NewWorkunit) Object %s not found in collection: %s", xerr.Error())
			return
		}
		clt.CwlVersion = job.CwlVersion

		if clt.CwlVersion == "" {
			err = fmt.Errorf("(NewWorkunit) CommandLineTool misses CwlVersion")
			return
		}
		workunit.CWL_workunit.CWL_tool = clt

		// ****** get inputs
		job_input_map := *job.CWL_collection.Job_input_map
		if job_input_map == nil {
			err = fmt.Errorf("(NewWorkunit) job.CWL_collection.Job_input_map is empty")
			return
		}
		//job_input_map := *job.CWL_collection.Job_input_map

		//job_input := *job.CWL_collection.Job_input

		workunit_input_map := make(map[string]cwl.CWLType) // also used for json

		fmt.Println("workflow_step.In:")
		spew.Dump(workflow_step.In)

		// 1. find all object source and Defaut
		// 2. make a map copy to be used in javaqscript, as "inputs"
		for _, input := range workflow_step.In {
			// input is a WorkflowStepInput

			id := input.Id

			cmd_id := path.Base(id)

			// get data from Source, Default or valueFrom

			if input.LinkMerge != nil {
				err = fmt.Errorf("(NewWorkunit) sorry, LinkMergeMethod not supported yet")
				return
			}

			if input.Source != nil {
				source_object_array := []cwl.CWLType{}
				//resolve pointers in source

				source_is_array := false

				source_as_string := ""
				source_as_array, source_is_array := input.Source.([]interface{})

				if source_is_array {
					fmt.Printf("source is a array: %s", spew.Sdump(input.Source))
					cwl_array := cwl.Array{}
					for _, src := range source_as_array { // usually only one
						fmt.Println("src: " + spew.Sdump(src))
						var src_str string
						var ok bool
						src_str, ok = src.(string)
						if !ok {
							err = fmt.Errorf("src is not a string")
							return
						}
						var job_obj cwl.CWLType
						job_obj, ok, err = getCWLSource(job_input_map, job, src_str)
						if err != nil {
							err = fmt.Errorf("(NewWorkunit) getCWLSource returns: %s", err.Error())
							return
						}
						if !ok {
							err = fmt.Errorf("(NewWorkunit) getCWLSource did not find output!!!")
						}
						source_object_array = append(source_object_array, job_obj)
						//cwl_array = append(cwl_array, obj)
					}

					workunit_input_map[cmd_id] = &cwl_array

				} else {
					fmt.Printf("source is NOT a array: %s", spew.Sdump(input.Source))
					var ok bool
					source_as_string, ok = input.Source.(string)
					if !ok {
						err = fmt.Errorf("(NewWorkunit) Cannot parse WorkflowStep source: %s", spew.Sdump(input.Source))
						return
					}

					var job_obj cwl.CWLType
					job_obj, ok, err = getCWLSource(job_input_map, job, source_as_string)
					if err != nil {
						err = fmt.Errorf("(NewWorkunit) getCWLSource returns: %s", err.Error())
						return
					}
					if !ok {
						err = fmt.Errorf("(NewWorkunit) getCWLSource did not find output!!!")
					}
					workunit_input_map[cmd_id] = job_obj

				}

			} else {

				if input.Default == nil {
					err = fmt.Errorf("(NewWorkunit) sorry, source and Default are missing")
					return
				}

				var default_value cwl.CWLType
				default_value, err = cwl.NewCWLType(cmd_id, input.Default)
				if err != nil {
					err = fmt.Errorf("(NewWorkunit) NewCWLTypeFromInterface(input.Default) returns: %s", err.Error())
					return
				}

				workunit_input_map[cmd_id] = default_value

			}
			// TODO

		}
		fmt.Println("workunit_input_map after first round:\n")
		spew.Dump(workunit_input_map)
		// 3. evaluate each ValueFrom field, update results

		for _, input := range workflow_step.In {
			if input.ValueFrom == "" {
				continue
			}

			id := input.Id
			cmd_id := path.Base(id)

			// from CWL doc: The self value of in the parameter reference or expression must be the value of the parameter(s) specified in the source field, or null if there is no source field.

			// #### Create VM ####
			vm := otto.New()

			// set "inputs"

			//func ToValue(value interface{}) (Value, error)

			//var inputs_value otto.Value
			//inputs_value, err = vm.ToValue(workunit_input_map)
			//if err != nil {
			//	return
			//}

			err = vm.Set("inputs", workunit_input_map)
			if err != nil {
				return
			}

			inputs_json, _ := json.Marshal(workunit_input_map)
			fmt.Printf("SET inputs=%s\n", inputs_json)

			js_self := workunit_input_map[cmd_id]
			err = vm.Set("self", js_self)
			if err != nil {
				return
			}
			self_json, _ := json.Marshal(js_self)
			fmt.Printf("SET self=%s\n", self_json)

			//fmt.Printf("input.ValueFrom=%s\n", input.ValueFrom)

			// evaluate $(...) ECMAScript expression
			reg := regexp.MustCompile(`\$\([\w.]+\)`)
			// CWL documentation: http://www.commonwl.org/v1.0/Workflow.html#Expressions

			parsed := input.ValueFrom.String()
			for {

				matches := reg.FindAll([]byte(parsed), -1)
				fmt.Printf("Matches: %d\n", len(matches))
				if len(matches) == 0 {
					break
				}
				for _, match := range matches {
					expression_string := bytes.TrimPrefix(match, []byte("$("))
					expression_string = bytes.TrimSuffix(expression_string, []byte(")"))

					javascript_function := fmt.Sprintf("(function(){\n return %s;\n})()", expression_string)
					fmt.Printf("%s\n", javascript_function)

					value, xerr := vm.Run(javascript_function)
					if xerr != nil {
						err = fmt.Errorf("Javascript complained: %s", xerr.Error())
						return
					}
					fmt.Println(reflect.TypeOf(value))

					value_str, xerr := value.ToString()
					if xerr != nil {
						err = fmt.Errorf("Cannot convert value to string: %s", xerr.Error())
						return
					}
					parsed = strings.Replace(parsed, string(match), value_str, 1)
				}

			}

			fmt.Printf("parsed: %s\n", parsed)

			// evaluate ${...} ECMAScript function body
			reg = regexp.MustCompile(`\$\{[\w.]+\}`)
			// CWL documentation: http://www.commonwl.org/v1.0/Workflow.html#Expressions

			//parsed = input.ValueFrom.String()

			matches := reg.FindAll([]byte(parsed), -1)
			fmt.Printf("Matches: %d\n", len(matches))
			if len(matches) == 0 {
				workunit_input_map[cmd_id] = cwl.NewStringFromstring(parsed)
			} else if len(matches) == 1 {
				match := matches[0]
				expression_string := bytes.TrimPrefix(match, []byte("${"))
				expression_string = bytes.TrimSuffix(expression_string, []byte("}"))

				javascript_function := fmt.Sprintf("(function(){\n %s \n})()", expression_string)
				fmt.Printf("%s\n", javascript_function)

				value, xerr := vm.Run(javascript_function)
				if xerr != nil {
					err = fmt.Errorf("Javascript complained: %s", xerr.Error())
					return
				}
				fmt.Printf("reflect.TypeOf(value): %s\n", reflect.TypeOf(value))

				var value_cwl cwl.CWLType
				value_cwl, err = cwl.NewCWLType("", value)
				if err != nil {
					err = fmt.Errorf("(NewWorkunit) Error parsing javascript VM result value: %s", err.Error())
					return
				}
				//value_str, xerr := value.ToString()
				//if xerr != nil {
				//	err = fmt.Errorf("Cannot convert value to string: %s", xerr.Error())
				//	return
				//}
				//parsed = strings.Replace(parsed, string(match), value_str, 1)

				//fmt.Printf("parsed: %s\n", parsed)

				//new_string := cwl.NewString(id, parsed)
				workunit_input_map[cmd_id] = value_cwl
			} else {
				err = fmt.Errorf("(NewWorkunit) ValueFrom contains more than one ECMAScript function body")
				return
			}

		}
		fmt.Println("workunit_input_map after second round:\n")
		spew.Dump(workunit_input_map)

		job_input := cwl.Job_document{}

		for elem_id, elem := range workunit_input_map {
			named_type := cwl.NewNamedCWLType(elem_id, elem)
			job_input = append(job_input, named_type)
		}

		workunit.CWL_workunit.Job_input = &job_input

		spew.Dump(job_input)

		workunit.CWL_workunit.OutputsExpected = &workflow_step.Out
		//spew.Dump(workflow_step.Out)
		//panic("done")

	}
	//panic("done")
	//spew.Dump(workunit.Cmd)
	//panic("done")

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

		task_id_array := strings.Split(work.Workunit_Unique_Identifier.TaskId, "/")
		task_name := ""
		if len(task_id_array) > 1 {
			task_name = strings.Join(task_id_array[1:], "/")
		} else {
			task_name = work.Workunit_Unique_Identifier.TaskId
		}

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
