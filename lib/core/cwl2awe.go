package core

import (
	"errors"
	"fmt"
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
	collection     *cwl.CWL_collection
	job            *Job
	AWE_tasks      *map[string]*Task
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

func CWL_input_check(job_input *cwl.Job_document, cwl_workflow *cwl.Workflow) (err error) {

	//job_input := *(collection.Job_input)

	job_input_map := job_input.GetMap() // map[string]CWLType

	for _, input := range cwl_workflow.Inputs {
		// input is a cwl.InputParameter object

		spew.Dump(input)

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
			input_obj_ref = cwl.NewNull(id_base)
			logger.Debug(3, "input %s not found, replace with Null object")
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
		has_type, xerr := cwl.TypeIsCorrect(expected_types, input_obj_ref)
		if xerr != nil {
			err = fmt.Errorf("(CWL_input_check) (B) HasInputParameterType returns: %s", xerr.Error())

			return
		}
		if !has_type {
			//if strings.ToLower(obj_type) != strings.ToLower(expected_types) {
			fmt.Printf("object found: ")
			spew.Dump(input_obj_ref)

			//for _, elem := range *expected_types {
			//	expected_types_str += "," + string(elem)
			//}
			fmt.Printf("cwl_workflow.Inputs")
			spew.Dump(cwl_workflow.Inputs)
			err = fmt.Errorf("Input %s has type %s, but this does not match the expected types)", id, input_type)
			return
		}

	}
	return
}

func CreateTasks(job *Job, workflow string, steps []cwl.WorkflowStep) (tasks []*Task, err error) {
	tasks = []*Task{}

	for s, _ := range steps {

		step := steps[s] // I could not do "_, step := range", that leas to very strange behaviour ?!??!

		//task_name := strings.Map(
		//	func(r rune) rune {
		//		if syntax.IsWordChar(r) || r == '/' || r == '-' { // word char: [0-9A-Za-z_]
		//			return r
		//		}
		//		return -1
		//	},
		//	step.Id)

		if !strings.HasPrefix(step.Id, "#") {
			err = fmt.Errorf("Workflow step name does not start with a #: %s", step.Id)
			return
		}
		task_name := step.Id

		if task_name == "" {
			err = fmt.Errorf("(CreateTasks) step_id is empty")
			return
		}

		//task_name := strings.TrimPrefix(step.Id, "#main/")
		//task_name = strings.TrimPrefix(task_name, "#")
		var awe_task *Task
		awe_task, err = NewTask(job, workflow, task_name)
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

func CWL2AWE(_user *user.User, files FormFiles, job_input *cwl.Job_document, cwl_workflow *cwl.Workflow, collection *cwl.CWL_collection) (job *Job, err error) {

	//CommandLineTools := collection.CommandLineTools

	// check that all expected workflow inputs exist and that they have the correct type
	logger.Debug(1, "CWL2AWE starting")

	err = CWL_input_check(job_input, cwl_workflow)
	if err != nil {
		err = fmt.Errorf("(CWL2AWE) CWL_input_check returned: %s", err.Error())
		return
	}

	//os.Exit(0)
	job = NewJob()
	job.setId()
	//job.CWL_workflow = cwl_workflow

	logger.Debug(1, "Job created")

	found_ShockRequirement := false
	for _, r := range cwl_workflow.Requirements { // TODO put ShockRequirement in Hints
		req, ok := r.(cwl.Requirement)
		if !ok {
			err = fmt.Errorf("not a requirement")
			return
		}
		switch req.GetClass() {
		case "ShockRequirement":
			sr, ok := req.(cwl.ShockRequirement)
			if !ok {
				err = fmt.Errorf("Could not assert ShockRequirement")
				return
			}

			job.ShockHost = sr.Host
			found_ShockRequirement = true

		}
	}

	if !found_ShockRequirement {
		//err = fmt.Errorf("ShockRequirement has to be provided in the workflow object")
		//return
		job.ShockHost = "http://shock:7445"

	}
	logger.Debug(1, "Requirements checked")

	// Once, job has been created, set job owner and add owner to all ACL's
	job.Acl.SetOwner(_user.Uuid)
	job.Acl.Set(_user.Uuid, acl.Rights{"read": true, "write": true, "delete": true})

	// TODO first check that all resources are available: local files and remote links

	main_wi := WorkflowInstance{Id: "", Inputs: *job_input}
	//new_wis := []WorkflowInstance{main_wi} // Not using AddWorkflowInstance to avoid mongo
	job.WorkflowInstances = make([]interface{}, 1)
	job.WorkflowInstances[0] = main_wi

	//if err != nil {
	//	return
	//}

	var tasks []*Task
	tasks, err = CreateTasks(job, "", cwl_workflow.Steps)
	if err != nil {
		return
	}

	job.Tasks = tasks

	_, err = job.Init()

	if err != nil {
		err = fmt.Errorf("job.Init() failed: %s", err.Error())
		return
	}
	logger.Debug(1, "Init called")

	err = job.Mkdir()
	if err != nil {
		err = errors.New("(CWL2AWE) error creating job directory, error=" + err.Error())
		return
	}

	err = job.UpdateFile(files, "cwl") // TODO that may not make sense. Check if I should store AWE job.
	if err != nil {
		err = errors.New("error in UpdateFile, error=" + err.Error())
		return
	}

	spew.Dump(job)

	logger.Debug(1, "job.Id: %s", job.Id)
	err = job.Save()
	if err != nil {
		err = errors.New("error in job.Save(), error=" + err.Error())
		return
	}
	return

}
