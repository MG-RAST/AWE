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

// Helper _
type Helper struct {
	unprocessedWS *map[string]*cwl.WorkflowStep
	processedWS   *map[string]*cwl.WorkflowStep
	//collection     *cwl.CWL_collection
	job      *Job
	AWETasks *map[string]*Task
}

func parseSourceString(source string, id string) (linkedStepName string, fieldname string, err error) {

	if !strings.HasPrefix(source, "#") {
		err = fmt.Errorf("source has to start with # (%s)", id)
		return
	}

	source = strings.TrimPrefix(source, "#")

	sourceArray := strings.Split(source, "/")

	switch len(sourceArray) {
	case 0:
		err = fmt.Errorf("source empty (%s)", id)
	case 1:
		fieldname = sourceArray[0]
	case 2:
		linkedStepName = sourceArray[0]
		fieldname = sourceArray[1]
	default:
		err = fmt.Errorf("source has too many fields (%s)", id)
	}

	return
}

// CWLInputCheck _
func CWLInputCheck(jobInput *cwl.Job_document, cwlWorkflow *cwl.Workflow, context *cwl.WorkflowContext) (jobInputNew *cwl.Job_document, err error) {

	//jobInput := *(collection.Job_input)

	jobInputMap := jobInput.GetMap() // map[string]CWLType

	for _, input := range cwlWorkflow.Inputs {
		// input is a cwl.InputParameter object

		//spew.Dump(input)

		id := input.Id
		logger.Debug(3, "(CWLInputCheck) Parsing workflow input %s", id)

		idBase := path.Base(id)
		expectedTypes := input.Type

		if len(expectedTypes) == 0 {
			err = fmt.Errorf("(CWLInputCheck) (len(expected_types) == 0 ")
			return
		}

		// find workflow input in jobInputMap
		inputObjRef, ok := jobInputMap[idBase] // returns CWLType
		if !ok {
			// not found, we can skip it it is optional anyway

			if input.Default != nil {
				logger.Debug(3, "input %s not found, replace with Default object", idBase)
				inputObjRef = input.Default

				//input_obj_ref_file, ok := input_obj_ref.(*cwl.File)

				jobInput = jobInput.Add(idBase, inputObjRef)
			} else {
				inputObjRef = cwl.NewNull()
				logger.Debug(3, "input %s not found, replace with Null object (no Default found)", idBase)
			}
		}

		if inputObjRef == nil {
			err = fmt.Errorf("(CWLInputCheck) input_obj_ref == nil")
			return
		}
		// Get type of CWL_Type we found
		inputType := inputObjRef.GetType()
		if inputType == nil {

			err = fmt.Errorf("(CWLInputCheck) input_type == nil %s", spew.Sdump(inputObjRef))
			return
		}
		//input_type_str := "unknown"
		logger.Debug(1, "(CWLInputCheck) input_type: %s (%s)", inputType, inputType.Type2String())

		// Check if type of input we have matches one of the allowed types
		hasType, xerr := cwl.TypeIsCorrect(expectedTypes, inputObjRef, context)
		if xerr != nil {
			err = fmt.Errorf("(CWLInputCheck) (B) HasInputParameterType returns: %s", xerr.Error())

			return
		}
		if !hasType {

			if inputObjRef.GetType() == cwl.CWLNull && input.Default != nil {
				// work-around to make sure ExpressionTool can use Default if it gets Null

			} else {

				//if strings.ToLower(obj_type) != strings.ToLower(expected_types) {
				fmt.Printf("object found: ")
				spew.Dump(inputObjRef)

				expectedTypesStr := ""
				for _, elem := range expectedTypes {
					expectedTypesStr += "," + elem.Type2String()
				}
				//fmt.Printf("cwl_workflow.Inputs")
				//spew.Dump(cwl_workflow.Inputs)
				err = fmt.Errorf("(CWLInputCheck) Input %s has type %s, but this does not match the expected types(%s)", id, inputType, expectedTypesStr)
				return
			}
		}

	}

	jobInputNew = jobInput

	return
}

// // CreateWorkflowTasks _
// func CreateWorkflowTasksDEPRECATED(job *Job, namePrefix string, steps []cwl.WorkflowStep, stepPrefix string, parentID *Task_Unique_Identifier) (tasks []*Task, err error) {
// 	tasks = []*Task{}

// 	if parentID == nil {
// 		err = fmt.Errorf("(CreateWorkflowTasks) parent_id == nil")
// 		return
// 	}

// 	entrypoint := job.Entrypoint

// 	if !strings.HasPrefix(namePrefix, entrypoint) {
// 		err = fmt.Errorf("(CreateWorkflowTasks) prefix_name does not start with _entrypoint %s: %s", entrypoint, namePrefix)
// 		return
// 	}

// 	for s := range steps {

// 		step := steps[s]

// 		if namePrefix == "" {
// 			err = fmt.Errorf("(CreateWorkflowTasks) name_prefix empty")
// 			return
// 		}

// 		if !strings.HasPrefix(step.ID, "#") {
// 			err = fmt.Errorf("(CreateWorkflowTasks) Workflow step name does not start with a #: %s", step.ID)
// 			return
// 		}

// 		fmt.Println("(CreateWorkflowTasks) step.Id: " + step.ID)

// 		// remove prefix
// 		taskName := path.Base(step.ID)
// 		//taskName := strings.TrimPrefix(step.Id, step_prefix)
// 		//taskName = strings.TrimSuffix(taskName, "/")
// 		//taskName = strings.TrimPrefix(taskName, "/")

// 		//fmt.Println("(CreateWorkflowTasks) taskName: " + taskName)
// 		//fmt.Println("(CreateWorkflowTasks) name_prefix: " + name_prefix)

// 		fmt.Println("(CreateWorkflowTasks) new task name will be: " + namePrefix + " / " + taskName)

// 		//taskName := strings.TrimPrefix(step.Id, #entrypoint/")
// 		//taskName = strings.TrimPrefix(taskName, "#")

// 		fmt.Printf("(CreateWorkflowTasks) creating task: %s %s\n", namePrefix, taskName)

// 		var aweTask *Task
// 		aweTask, err = NewTask(job, "DEPRECATED", namePrefix, taskName)
// 		if err != nil {
// 			err = fmt.Errorf("(CreateWorkflowTasks) NewTask returned: %s", err.Error())
// 			return
// 		}

// 		//aweTask.WorkflowParent = parent_id

// 		if step.ID == "" {
// 			err = fmt.Errorf("(CreateWorkflowTasks) step.Id empty")
// 			return
// 		}

// 		aweTask.WorkflowStep = &step
// 		//spew.Dump(step)
// 		tasks = append(tasks, aweTask)

// 	}
// 	return
// }

// CWL2AWE _
func CWL2AWE(_user *user.User, files FormFiles, jobInput *cwl.Job_document, cwlWorkflow *cwl.Workflow, entrypoint string, context *cwl.WorkflowContext) (job *Job, err error) {

	// check that all expected workflow inputs exist and that they have the correct type
	logger.Debug(1, "(CWL2AWE) CWL2AWE starting..")
	defer logger.Debug(1, "(CWL2AWE) CWL2AWE leaving...")

	var jobInputNew *cwl.Job_document
	jobInputNew, err = CWLInputCheck(jobInput, cwlWorkflow, context)
	if err != nil {
		err = fmt.Errorf("(CWL2AWE) CWLInputCheck returned: %s", err.Error())
		return
	}

	//os.Exit(0)
	job = NewJob()

	job.setID()

	job.WorkflowContext = context
	//job.CWL_workflow = cwl_workflow

	logger.Debug(1, "(CWL2AWE) Job created")

	foundShockRequirement := false
	if cwlWorkflow.Requirements != nil {
		for _, r := range cwlWorkflow.Requirements { // TODO put ShockRequirement in Hints
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
				foundShockRequirement = true

			}
		}
	}

	if !foundShockRequirement {
		err = fmt.Errorf("(CWL2AWE) ShockRequirement has to be provided in the workflow object")
		return
		//job.ShockHost = "http://shock:7445" // TODO make this different

	}
	logger.Debug(1, "(CWL2AWE) Requirements checked")

	// Once, job has been created, set job owner and add owner to all ACL's
	job.ACL.SetOwner(_user.Uuid)
	job.ACL.Set(_user.Uuid, acl.Rights{"read": true, "write": true, "delete": true})

	logger.Debug(1, "(CWL2AWE) ACLs set")
	// TODO first check that all resources are available: local files and remote links

	// *** create WorkflowInstance

	var wi *WorkflowInstance
	wi, err = NewWorkflowInstance(entrypoint, job.ID, cwlWorkflow.ID, job, "") // Not using AddWorkflowInstance to avoid mongo
	if err != nil {
		err = fmt.Errorf("(CWL2AWE) NewWorkflowInstance returned: %s", err.Error())
		return
	}

	wi.Inputs = *jobInputNew
	logger.Debug(1, "(CWL2AWE) WorkflowInstance %s created", entrypoint)

	// create path
	//for i, _ := range wi.Inputs {
	//	new_id := cwl_workflow.Id + "/" + wi.Inputs[i].Id
	//wi.Inputs[i].Id = new_id
	//err = context.Add(new_id, wi.Inputs[i].Value, "CWL2AWE")
	//if err != nil {
	//	err = fmt.Errorf("(CWL2AWE) context.Add returned: %s", err.Error())
	//	return
	//}
	//}

	err = wi.Insert(false)
	if err != nil {
		err = fmt.Errorf("(CWL2AWE) wi.Save returned: %s", err.Error())
		return
	}

	logger.Debug(1, "(CWL2AWE) wi saved")

	err = job.AddWorkflowInstance(wi, DbSyncFalse, false) // adding _root
	if err != nil {
		err = fmt.Errorf("(CWL2AWE) AddWorkflowInstance returned: %s", err.Error())
		return
	}

	job.Root = wi.ID

	logger.Debug(1, "(CWL2AWE) wi added")

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

	logger.Debug(1, "(CWL2AWE) job.Id: %s", job.ID)
	err = job.Save()
	if err != nil {
		err = errors.New("(CWL2AWE) error in job.Save(), error=" + err.Error())
		return
	}
	return

}
