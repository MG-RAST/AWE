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
	requirements "github.com/MG-RAST/AWE/lib/core/cwl/requirements"
	"path"
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

	job_input_map := job_input.GetMap()

	for _, input := range cwl_workflow.Inputs {
		// input is a cwl.InputParameter object

		spew.Dump(input)

		id := input.Id
		logger.Debug(3, "Parsing workflow input %s", id)
		//if strings.HasPrefix(id, "#") {
		//	workflow_prefix := cwl_workflow.Id + "/"
		//	if strings.HasPrefix(id, workflow_prefix) {
		//		id = strings.TrimPrefix(id, workflow_prefix)
		//		logger.Debug(3, "using new id : %s", id)
		//	} else {
		//		err = fmt.Errorf("prefix of id %s does not match workflow id %s with workflow_prefix: %s", id, cwl_workflow.Id, workflow_prefix)
		//		return
		//	}
		//}

		id_base := path.Base(id)
		expected_types := input.Type

		if len(expected_types) == 0 {
			err = fmt.Errorf("len(expected_types) == 0 ")
			return
		}

		obj_ref, ok := job_input_map[id_base]
		if !ok {
			// not found, we can skip it it is optional anyway

			has_type, xerr := cwl.HasInputParameterType(expected_types, cwl.CWL_null)
			if xerr != nil {
				err = fmt.Errorf("(CWL_input_check) (A) HasInputParameterType returns: %s", xerr)
				return
			}
			if has_type { // null type means parameter is optional
				continue
			}

			// well, this was not optional and not found

			fmt.Printf("-------expected_types:")
			spew.Dump(expected_types)

			fmt.Printf("-------job_input_map:")
			spew.Dump(job_input_map)
			panic("uhoh")
			err = fmt.Errorf("value for workflow input \"%s\" not found", id)
			fmt.Println(err.Error())
			panic("uhoh")
			return

		}

		//obj := *obj_ref
		obj_type := obj_ref.GetClass()
		logger.Debug(1, "obj_type: %s", obj_type)
		has_type, xerr := cwl.HasInputParameterType(expected_types, obj_type)
		if xerr != nil {
			err = fmt.Errorf("(CWL_input_check) (B) HasInputParameterType returns: %s", xerr)

			return
		}
		if !has_type {
			//if strings.ToLower(obj_type) != strings.ToLower(expected_types) {
			fmt.Printf("object found: ")
			spew.Dump(obj_ref)

			//for _, elem := range *expected_types {
			//	expected_types_str += "," + string(elem)
			//}
			fmt.Printf("cwl_workflow.Inputs")
			spew.Dump(cwl_workflow.Inputs)
			err = fmt.Errorf("Input %s got %s)", id, obj_type)
			return
		}

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
			sr, ok := req.(requirements.ShockRequirement)
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

	//helper := Helper{}

	//processed_ws := make(map[string]*cwl.WorkflowStep)
	//unprocessed_ws := make(map[string]*cwl.WorkflowStep)
	//awe_tasks := make(map[string]*Task)
	//helper.processed_ws = &processed_ws
	//helper.unprocessed_ws = &unprocessed_ws
	//helper.collection = collection
	//helper.job = job
	//helper.AWE_tasks = &awe_tasks

	for _, step := range cwl_workflow.Steps {
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
		awe_task := NewTask(job, task_name)
		awe_task.WorkflowStep = &step
		job.Tasks = append(job.Tasks, awe_task)
	}

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
