package core

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/MG-RAST/AWE/lib/acl"
	"github.com/MG-RAST/AWE/lib/core/cwl"

	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/user"
	"github.com/davecgh/go-spew/spew"
	//aw_sequences"os"
	"path"
	//"reflect"
	//"strconv"
	//"regexp/syntax"
	"strings"
)

type Helper struct {
	unprocessed_ws *map[string]*cwl.WorkflowStep
	processed_ws   *map[string]*cwl.WorkflowStep
	//collection     *cwl.CWL_collection
	job       *Job
	AWE_tasks *map[string]*Task
}

func parseSourceString(source string, id string) (linked_step_name string, fieldname string, err error) {

	if !strings.HasPrefix(source, "#") {
		err = fmt.Errorf("source has to start with # (%s)", id)
		return
	}

	source = strings.TrimPrefix(source, "#")

	source_array := strings.Split(source, "/")

	switch len(source_array) {
	case 0:
		err = fmt.Errorf("source empty (%s)", id)
	case 1:
		fieldname = source_array[0]
	case 2:
		linked_step_name = source_array[0]
		fieldname = source_array[1]
	default:
		err = fmt.Errorf("source has too many fields (%s)", id)
	}

	return
}

func CWL_input_check(job_input *cwl.Job_document, cwl_workflow *cwl.Workflow, context *cwl.WorkflowContext) (job_input_new *cwl.Job_document, err error) {

	//job_input := *(collection.Job_input)

	job_input_map := job_input.GetMap() // map[string]CWLType

	for _, input := range cwl_workflow.Inputs {
		// input is a cwl.InputParameter object

		//spew.Dump(input)

		id := input.Id
		logger.Debug(3, "(CWL_input_check) Parsing workflow input %s", id)

		id_base := path.Base(id)
		expected_types := input.Type

		if len(expected_types) == 0 {
			err = fmt.Errorf("(CWL_input_check) (len(expected_types) == 0 ")
			return
		}

		// find workflow input in job_input_map
		input_obj_ref, ok := job_input_map[id_base] // returns CWLType
		if !ok {
			// not found, we can skip it it is optional anyway

			if input.Default != nil {
				logger.Debug(3, "input %s not found, replace with Default object", id_base)
				input_obj_ref = input.Default

				//input_obj_ref_file, ok := input_obj_ref.(*cwl.File)

				job_input = job_input.Add(id_base, input_obj_ref)
			} else {
				input_obj_ref = cwl.NewNull()
				logger.Debug(3, "input %s not found, replace with Null object (no Default found)", id_base)
			}
		}

		if input_obj_ref == nil {
			err = fmt.Errorf("(CWL_input_check) input_obj_ref == nil")
			return
		}
		// Get type of CWL_Type we found
		input_type := input_obj_ref.GetType()
		if input_type == nil {

			err = fmt.Errorf("(CWL_input_check) input_type == nil %s", spew.Sdump(input_obj_ref))
			return
		}
		//input_type_str := "unknown"
		logger.Debug(1, "(CWL_input_check) input_type: %s (%s)", input_type, input_type.Type2String())

		// Check if type of input we have matches one of the allowed types
		has_type, xerr := cwl.TypeIsCorrect(expected_types, input_obj_ref, context)
		if xerr != nil {
			err = fmt.Errorf("(CWL_input_check) (B) HasInputParameterType returns: %s", xerr.Error())

			return
		}
		if !has_type {

			if input_obj_ref.GetType() == cwl.CWL_null && input.Default != nil {
				// work-around to make sure ExpressionTool can use Default if it gets Null

			} else {

				//if strings.ToLower(obj_type) != strings.ToLower(expected_types) {
				fmt.Printf("object found: ")
				spew.Dump(input_obj_ref)

				expected_types_str := ""
				for _, elem := range expected_types {
					expected_types_str += "," + elem.Type2String()
				}
				//fmt.Printf("cwl_workflow.Inputs")
				//spew.Dump(cwl_workflow.Inputs)
				err = fmt.Errorf("(CWL_input_check) Input %s has type %s, but this does not match the expected types(%s)", id, input_type, expected_types_str)
				return
			}
		}

	}

	job_input_new = job_input

	return
}

func CreateWorkflowTasks(job *Job, name_prefix string, steps []cwl.WorkflowStep, step_prefix string) (tasks []*Task, err error) {
	tasks = []*Task{}

	if !strings.HasPrefix(name_prefix, "_root") {
		err = fmt.Errorf("prefix_name does not start with _entrypoint: %s", name_prefix)
		return
	}

	for s, _ := range steps {

		step := steps[s]

		if !strings.HasPrefix(step.Id, "#") {
			err = fmt.Errorf("Workflow step name does not start with a #: %s", step.Id)
			return
		}

		fmt.Println("step.Id: " + step.Id)

		// remove prefix
		task_name := strings.TrimPrefix(step.Id, step_prefix)
		task_name = strings.TrimSuffix(task_name, "/")
		task_name = strings.TrimPrefix(task_name, "/")

		fmt.Println("task_name1: " + task_name)
		fmt.Println("name_prefix: " + name_prefix)

		fmt.Println("new task name will be: " + name_prefix + "/" + task_name)

		//task_name := strings.TrimPrefix(step.Id, "#main/")
		//task_name = strings.TrimPrefix(task_name, "#")

		fmt.Printf("(CreateTasks) creating task: %s %s\n", name_prefix, task_name)

		var awe_task *Task
		awe_task, err = NewTask(job, name_prefix, task_name)
		if err != nil {
			err = fmt.Errorf("(CreateTasks) NewTask returned: %s", err.Error())
			return
		}

		awe_task.WorkflowStep = &step
		//spew.Dump(step)
		tasks = append(tasks, awe_task)

	}
	return
}

func CWL2AWE(_user *user.User, files FormFiles, job_input *cwl.Job_document, cwl_workflow *cwl.Workflow, context *cwl.WorkflowContext) (job *Job, err error) {

	// check that all expected workflow inputs exist and that they have the correct type
	logger.Debug(1, "(CWL2AWE) CWL2AWE starting")

	var job_input_new *cwl.Job_document
	job_input_new, err = CWL_input_check(job_input, cwl_workflow, context)
	if err != nil {
		err = fmt.Errorf("(CWL2AWE) CWL_input_check returned: %s", err.Error())
		return
	}

	//os.Exit(0)
	job = NewJob()

	job.setId()

	job.WorkflowContext = context
	//job.CWL_workflow = cwl_workflow

	logger.Debug(1, "(CWL2AWE) Job created")

	found_ShockRequirement := false
	if cwl_workflow.Requirements != nil {
		for _, r := range cwl_workflow.Requirements { // TODO put ShockRequirement in Hints
			req, ok := r.(cwl.Requirement)
			if !ok {
				err = fmt.Errorf("(CWL2AWE) not a requirement")
				return
			}
			switch req.GetClass() {
			case "ShockRequirement":
				sr, ok := r.(*cwl.ShockRequirement)
				if !ok {
					err = fmt.Errorf("(CWL2AWE) Could not assert ShockRequirement (type: %s)", reflect.TypeOf(r))
					return
				}

				job.ShockHost = sr.Shock_api_url
				found_ShockRequirement = true

			}
		}
	}

	if !found_ShockRequirement {
		err = fmt.Errorf("(CWL2AWE) ShockRequirement has to be provided in the workflow object")
		return
		//job.ShockHost = "http://shock:7445" // TODO make this different

	}
	logger.Debug(1, "(CWL2AWE) Requirements checked")

	// Once, job has been created, set job owner and add owner to all ACL's
	job.Acl.SetOwner(_user.Uuid)
	job.Acl.Set(_user.Uuid, acl.Rights{"read": true, "write": true, "delete": true})

	// TODO first check that all resources are available: local files and remote links

	var wi *WorkflowInstance
	wi, err = NewWorkflowInstance("_root", job.Id, cwl_workflow.Id, *job_input_new, job) // Not using AddWorkflowInstance to avoid mongo
	if err != nil {
		err = fmt.Errorf("(CWL2AWE) NewWorkflowInstance returned: %s", err.Error())
		return
	}

	var tasks []*Task
	tasks, err = CreateWorkflowTasks(job, "_root", cwl_workflow.Steps, cwl_workflow.Id)
	if err != nil {
		err = fmt.Errorf("(CWL2AWE) CreateWorkflowTasks returned: %s", err.Error())
		return
	}

	if len(tasks) == 0 {
		err = fmt.Errorf("(CWL2AWE) no tasks ?")
		return
	}

	//job.Tasks = tasks
	err = wi.SetTasks(tasks, "db_sync_no")
	if err != nil {
		err = fmt.Errorf("(CWL2AWE) wi.SetTasks returned: %s", err.Error())
		return
	}

	err = job.AddWorkflowInstance(wi) // adding _root
	if err != nil {
		err = fmt.Errorf("(CWL2AWE) AddWorkflowInstance returned: %s", err.Error())
		return
	}

	err = wi.Save(true)
	if err != nil {
		err = fmt.Errorf("(CWL2AWE) wi.Save returned: %s", err.Error())
		return
	}

	_, err = job.Init()

	if err != nil {
		err = fmt.Errorf("(CWL2AWE) job.Init() failed: %s", err.Error())
		return
	}
	logger.Debug(1, "(CWL2AWE) Init called")

	err = job.Mkdir()
	if err != nil {
		err = errors.New("(CWL2AWE) error creating job directory, error=" + err.Error())
		return
	}

	err = job.UpdateFile(files, "cwl") // TODO that may not make sense. Check if I should store AWE job.
	if err != nil {
		err = errors.New("(CWL2AWE) error in UpdateFile, error=" + err.Error())
		return
	}

	//spew.Dump(job)

	//panic("done")

	logger.Debug(1, "job.Id: %s", job.Id)
	err = job.Save()
	if err != nil {
		err = errors.New("(CWL2AWE) error in job.Save(), error=" + err.Error())
		return
	}
	return

}
