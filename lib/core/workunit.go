package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
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
	Workunit_Unique_Identifier `bson:",inline" json:",inline" mapstructure:",squash"`
	Id                         string            `bson:"id,omitempty" json:"id,omitempty"`     // global identifier: jobid_taskid_rank (for backwards coompatibility only)
	WuId                       string            `bson:"wuid,omitempty" json:"wuid,omitempty"` // deprecated !
	Info                       *Info             `bson:"info,omitempty" json:"info,omitempty"`
	Inputs                     []*IO             `bson:"inputs,omitempty" json:"inputs,omitempty"`
	Outputs                    []*IO             `bson:"outputs,omitempty" json:"outputs,omitempty"`
	Predata                    []*IO             `bson:"predata,omitempty" json:"predata,omitempty"`
	Cmd                        *Command          `bson:"cmd,omitempty" json:"cmd,omitempty"`
	TotalWork                  int               `bson:"totalwork,omitempty" json:"totalwork,omitempty"`
	Partition                  *PartInfo         `bson:"part,omitempty" json:"part,omitempty"`
	State                      string            `bson:"state,omitempty" json:"state,omitempty"`
	Failed                     int               `bson:"failed,omitempty" json:"failed,omitempty"`
	CheckoutTime               time.Time         `bson:"checkout_time,omitempty" json:"checkout_time,omitempty"`
	Client                     string            `bson:"client,omitempty" json:"client,omitempty"`
	ComputeTime                int               `bson:"computetime,omitempty" json:"computetime,omitempty"`
	ExitStatus                 int               `bson:"exitstatus,omitempty" json:"exitstatus,omitempty"` // Linux Exit Status Code (0 is success)
	Notes                      []string          `bson:"notes,omitempty" json:"notes,omitempty"`
	UserAttr                   map[string]string `bson:"userattr,omitempty" json:"userattr,omitempty"`
	WorkPath                   string            // this is the working directory. If empty, it will be computed.
	WorkPerf                   *WorkPerf
	CWL                        *CWL_workunit
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

		workunit.CWL = &CWL_workunit{}

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

		workunit.CWL.CWL_tool = clt

		// ****** get inputs
		job_input := *job.CWL_collection.Job_input
		if job.CWL_collection.Job_input_map == nil {
			err = fmt.Errorf("(NewWorkunit) job.CWL_collection.Job_input_map is empty")
			return
		}
		job_input_map := *job.CWL_collection.Job_input_map

		spew.Dump(workflow_step.In)
		for _, input := range workflow_step.In {
			// input is a WorkflowStepInput

			id := input.Id

			cmd_id := path.Base(id)

			// get data from Source, Default or valueFrom

			if input.LinkMerge != nil {
				err = fmt.Errorf("(NewWorkunit) sorry, LinkMergeMethod not supported yet")
				return
			}

			source_object_array := []cwl_types.CWLType{}
			//resolve pointers in source
			for _, src := range input.Source {
				// src is a string, an id to another cwl object (workflow input of step output)

				src_base := path.Base(src)

				job_obj, ok := job_input_map[src_base]
				if !ok {
					fmt.Printf("%s not found in \n", src_base)
				} else {
					fmt.Printf("%s found in job_input!!!\n", src_base)
					source_object_array = append(source_object_array, job_obj)
					continue
				}

				coll_obj, xerr := job.CWL_collection.Get(src)
				if xerr != nil {
					fmt.Printf("%s not found in CWL_collection\n", src)
				} else {
					fmt.Printf("%s found in CWL_collection!!!\n", src)
					//source_object_array = append(source_object_array, coll_obj)
					continue
				}
				_ = coll_obj
				err = fmt.Errorf("Source object %s not found", src)
				return

			}

			// input.Default  The default value for this parameter to use if either there is no source field, or the value produced by the source is null. The default must be applied prior to scattering or evaluating valueFrom.

			if len(input.Source) == 1 {
				job_input_map[cmd_id] = source_object_array[0]
				object := source_object_array[0]
				fmt.Println("WORLD")
				spew.Dump(object)

				file, ok := object.(*cwl_types.File)
				if ok {
					fmt.Println("A FILE")
					fmt.Printf("%+v\n", *file)

				}

			} else if len(input.Source) > 1 {
				cwl_array := cwl_types.Array{}
				for _, obj := range source_object_array {
					cwl_array.Add(obj)
				}
				job_input_map[cmd_id] = &cwl_array
			} else {
				if input.Default != nil {
					err = fmt.Errorf("(NewWorkunit) sorry, Default not supported yet")
					return
				}
			}

			if input.ValueFrom != "" {

				// from CWL doc: The self value of in the parameter reference or expression must be the value of the parameter(s) specified in the source field, or null if there is no source field.

				vm := otto.New()
				//TODO vm.Set("input", 11)

				if len(source_object_array) == 1 {
					obj := source_object_array[0]

					vm.Set("self", obj.String())
					fmt.Printf("SET self=%s\n", input.Source[0])
				} else if len(input.Source) > 1 {

					source_b, xerr := json.Marshal(input.Source)
					if xerr != nil {
						err = fmt.Errorf("(NewWorkunit) cannot marshal source: %s", xerr.Error())
						return
					}
					vm.Set("self", string(source_b[:]))
					fmt.Printf("SET self=%s\n", string(source_b[:]))
				}
				fmt.Printf("input.ValueFrom=%s\n", input.ValueFrom)

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
							err = fmt.Errorf("Cannot convert value to string: %s", xerr)
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

				for {

					matches := reg.FindAll([]byte(parsed), -1)
					fmt.Printf("Matches: %d\n", len(matches))
					if len(matches) == 0 {
						break
					}
					for _, match := range matches {
						expression_string := bytes.TrimPrefix(match, []byte("${"))
						expression_string = bytes.TrimSuffix(expression_string, []byte("}"))

						javascript_function := fmt.Sprintf("(function(){\n %s \n})()", expression_string)
						fmt.Printf("%s\n", javascript_function)

						value, xerr := vm.Run(javascript_function)
						if xerr != nil {
							err = fmt.Errorf("Javascript complained: %s", xerr.Error())
							return
						}
						fmt.Println(reflect.TypeOf(value))

						value_str, xerr := value.ToString()
						if xerr != nil {
							err = fmt.Errorf("Cannot convert value to string: %s", xerr)
							return
						}
						parsed = strings.Replace(parsed, string(match), value_str, 1)
					}

				}

				fmt.Printf("parsed: %s\n", parsed)

				new_string := cwl_types.NewString(id, parsed)
				job_input_map[cmd_id] = new_string
				// TODO does this have to be storted in job_input ???

				//err = fmt.Errorf("(NewWorkunit) sorry, ValueFrom not supported yet")

			}
			workunit.CWL.Job_input = &job_input
			spew.Dump(job_input)

		}

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
