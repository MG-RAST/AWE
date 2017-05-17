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
	"strconv"
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

	source_array := strings.Split(source, ".")

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

func createAweTask(helper *Helper, cwl_tool *cwl.CommandLineTool, cwl_step *cwl.WorkflowStep, awe_task *Task) (err error) {

	// valueFrom is a StepInputExpressionRequirement
	// http://www.commonwl.org/v1.0/Workflow.html#StepInputExpressionRequirement

	//collection := helper.collection
	//unprocessed_ws := helper.unprocessed_ws
	processed_ws := helper.processed_ws

	local_collection := cwl.NewCWL_collection()
	//workflowStepInputMap := make(map[string]cwl.WorkflowStepInput)

	linked_IO := make(map[string]*IO)

	spew.Dump(cwl_tool)
	spew.Dump(cwl_step)

	fmt.Println("=============================================== variables")
	// collect variables in local_collection
	for _, workflow_step_input := range cwl_step.In {
		// input is a cwl.WorkflowStepInput

		spew.Dump(workflow_step_input)
		id := workflow_step_input.Id

		if id == "" {
			err = fmt.Errorf("id is empty")
			return
		}

		fmt.Println("//////////////// step input: " + id)

		if len(workflow_step_input.Source) > 0 {

			if len(workflow_step_input.Source) == 1 {
				source := workflow_step_input.Source[0] // #fieldname or #stepname.fieldname

				var linked_step_name string
				var fieldname string
				linked_step_name, fieldname, err = parseSourceString(source, id)
				if err != nil {
					return
				}

				// find fieldname, may refer to other task, recurse in creation of that other task if necessary
				var linked_step cwl.WorkflowStep
				if linked_step_name != "" {
					this_linked_step, ok := (*processed_ws)[linked_step_name]
					if !ok {

						// recurse into depending step
						err = cwl_step_2_awe_task(helper, linked_step_name)
						if err != nil {
							err = fmt.Errorf("source empty (%s) (%s)", id, err.Error())
							return
						}

						this2_linked_step, ok := (*processed_ws)[linked_step_name]
						if !ok {
							err = fmt.Errorf("linked_step %s not found after processing", linked_step_name)
							return
						}
						linked_step = *this2_linked_step
					} else {
						linked_step = *this_linked_step
					}
					// linked_step is defined now.
					_ = linked_step

					// search output of linked_step

					linked_awe_task, ok := (*helper.AWE_tasks)[linked_step_name]
					if !ok {
						err = fmt.Errorf("Could not find AWE task for linked step %s", linked_step_name)
						return
					}
					linked_task_id := linked_awe_task.Id

					linked_filename := ""
					for _, awe_output := range linked_awe_task.Outputs {
						// awe_output is an AWE IO{} object
						if awe_output.Name == fieldname {
							linked_filename = awe_output.FileName
							break
						}
					}

					if linked_filename == "" {
						err = fmt.Errorf("Output %s not found", fieldname)
						return
					}
					//linked_output, err = linked_step.GetOutput(fieldname)
					//if err != nil {
					//	return
					//}

					// linked_output.Id provides
					linked_IO[id] = &IO{Origin: linked_task_id, FileName: linked_filename, Name: id}

				} else {
					// use workflow input as input

					var obj *cwl.CWL_object
					obj, err = (*helper.collection).Get(fieldname)
					if err != nil {
						return
					}

					obj_local := *obj
					obj_local.SetId(id)
					fmt.Printf("ADDEDb %s\n", id)
					fmt.Printf("ADDED2b %s\n", obj_local.GetId())
					err = local_collection.Add(obj_local)
					if err != nil {
						return
					}

				}

			} else {
				err = fmt.Errorf("MultipleInputFeatureRequirement not supported yet (id=%s)", id)
				return
			}

		} else {

			var obj *cwl.CWL_object
			obj, err = workflow_step_input.GetObject(helper.collection) // from ValueFrom or Default fields
			if err != nil {
				return
			}
			obj_local := *obj

			//obj_int, ok := obj_local.(cwl.Int)
			//if ok {
			//	fmt.Printf("obj_int.Id: %s\n", obj_int.Id)
			//	obj_int.Id = "test"
			//	fmt.Printf("obj_int.Id: %s\n", obj_int.Id)
			//	fmt.Printf("obj_int.Value: %d\n", obj_int.Value)
			//	spew.Dump(obj_int)
			//	obj_int.SetId(id)
			//	fmt.Printf("obj_int]]]]]]] %s\n", obj_int.GetId())
			//}

			spew.Dump(obj_local)
			obj_local.SetId(id)
			spew.Dump(obj_local)
			fmt.Printf("ADDED %s\n", id)
			fmt.Printf("ADDED2 %s\n", obj_local.GetId())
			err = local_collection.Add(obj_local) // add object with local input name
			if err != nil {
				return
			}
		}

		//workflowStepInputMap[id] = input
		// use content of input instead ": local_collection.Add(input)
		//fmt.Println("ADD local_collection.WorkflowStepInputs")
		//spew.Dump(local_collection.WorkflowStepInputs)
	}
	//fmt.Println("workflowStepInputMap:")
	//spew.Dump(workflowStepInputMap)

	// copy cwl_tool to AWE via step
	command_line := ""
	fmt.Println("=============================================== expected inputs")
	for _, expected_input := range cwl_tool.Inputs {
		// input is a cwl.CommandInputParameter

		id := expected_input.Id
		_ = id

		if len(expected_input.Type) > 1 {
			err = fmt.Errorf("Not yet supported: len(expected_input.Type) > 1")
			return
		}
		expected_input_parameter_type_0 := expected_input.Type[0]
		_ = expected_input_parameter_type_0
		//input_optional := strings.HasSuffix(expected_input_type_0, "?") TODO
		//expected_input_type := strings.ToLower(strings.TrimSuffix(expected_input_type_0, "?"))TODO
		//_ = input_optional
		//_ = expected_input_type

		// TODO lookup id in workflow step input

		//fmt.Println("local_collection.WorkflowStepInputs")
		//spew.Dump(local_collection.WorkflowStepInputs)
		//workflow_step_input, ok := workflowStepInputMap[id]
		//actual_input, xerr := local_collection.Get(id)
		//if xerr != nil {
		//	err = fmt.Errorf("%s not found in workflowStepInputMap", id)
		//	return
		//}
		//fmt.Println("workflow_step_input: ")
		//spew.Dump(workflow_step_input)
		input_string := ""
		var obj *cwl.CWL_object
		obj, err = local_collection.Get(id)
		if err != nil {

			io_struct, ok := linked_IO[id]
			if !ok {
				err = fmt.Errorf("%s not found in local_collection or linked_IO", id)

				fmt.Println("-------------------------------------------------")
				spew.Dump(local_collection)
				spew.Dump(linked_IO)

				return
			}

			// TODO
			// io_struct.DataToken = ..

			awe_task.Inputs = append(awe_task.Inputs, io_struct)

		} else {

			obj_class := (*obj).GetClass()
			switch obj_class {

			case "File":

				var file_obj *cwl.File
				file_obj, err = local_collection.GetFile(id)
				if err != nil {
					err = fmt.Errorf("%s not found in local_collection ", id)
					return
				}

				// TODO HERE HERE HERE HERE HERE HERE HERE HERE HERE HERE HERE HERE HERE
				input_string = file_obj.Basename

				awe_input := NewIO()
				awe_input.FileName = file_obj.Basename
				awe_input.Name = file_obj.Id
				awe_input.Host = file_obj.Host
				awe_input.Node = file_obj.Node
				//awe_input.Url=input_file.
				awe_input.DataToken = file_obj.Token

				awe_task.Inputs = append(awe_task.Inputs, awe_input)
			case "String":

				var string_obj *cwl.String
				string_obj, err = local_collection.GetString(id)
				if err != nil {
					err = fmt.Errorf("%s not found in local_collection ", id)
					return
				}
				input_string = string_obj.String()
			case "Int":

				var int_obj *cwl.Int
				int_obj, err = local_collection.GetInt(id)
				if err != nil {
					err = fmt.Errorf("%s not found in local_collection ", id)
					return
				}
				input_string = int_obj.String()
			default:

				err = fmt.Errorf("not implemented yet (%s)", obj_class)

			}

		}

		input_binding := expected_input.InputBinding

		//if input_binding != nil {
		prefix := input_binding.Prefix
		if prefix != "" {

			if command_line != "" {
				command_line = command_line + " "
			}
			command_line = command_line + prefix + " " + input_string
		}

		//}
		// Id:  #fieldname or #stepname.fieldname
		//switch input.Source {
		//case

		//}
		fmt.Println("command_line: ", command_line)
	}

	awe_task.Cmd.Args = command_line
	awe_task.Cmd.Name = cwl_tool.BaseCommand

	for _, requirement := range cwl_tool.Hints {
		class := requirement.GetClass()
		switch class {
		case "DockerRequirement":
			dr := requirement.(cwl.DockerRequirement)
			if dr.DockerPull != "" {
				awe_task.Cmd.Dockerimage = dr.DockerPull
			}

			if dr.DockerLoad != "" {
				err = fmt.Errorf("DockerRequirement.DockerLoad not supported yet")
				return
			}
			if dr.DockerFile != "" {
				err = fmt.Errorf("DockerRequirement.DockerFile not supported yet")
				return
			}
			if dr.DockerImport != "" {
				err = fmt.Errorf("DockerRequirement.DockerImport not supported yet")
				return
			}
			if dr.DockerImageId != "" {
				err = fmt.Errorf("DockerRequirement.DockerImageId not supported yet")
				return
			}
		default:
			err = fmt.Errorf("Requirement \"%s\" not supported", class)
			return
		}
	}

	// collect tool outputs
	tool_outputs := make(map[string]*cwl.CommandOutputParameter)
	for _, output := range cwl_tool.Outputs {
		tool_outputs[output.Id] = &output
	}

	for _, output := range cwl_step.Out {
		// output is a WorkflowStepOutput
		output_id := output.Id

		// lookup glob of CommandLine tool
		cmd_out_param, ok := tool_outputs[output_id]
		if !ok {
			err = fmt.Errorf("output_id %s not found in tool_outputs", output_id)
			return
		}

		awe_output := NewIO()
		awe_output.Name = output_id
		glob_array := cmd_out_param.OutputBinding.Glob
		switch len(glob_array) {
		case 0:
			err = fmt.Errorf("output_id %s glob empty", output_id)
			return
		case 1:
			glob := glob_array[0]

			fmt.Println("READ local_collection.WorkflowStepInputs")
			spew.Dump(local_collection.WorkflowStepInputs)

			glob_evaluated := local_collection.Evaluate(glob.String())
			fmt.Printf("glob: %s -> %s", glob.String(), glob_evaluated)
			// TODO evalue glob from workflowStepInputMap
			//step_input, xerr := local_collection.Get(id)
			//if xerr != nil {
			//	err = xerr
			//	return
			//}
			//blubb := step_input.Default.String()

			awe_output.FileName = glob_evaluated
		default:
			err = fmt.Errorf("output_id %s too many globs, not supported yet", output_id)
			return
		}

		if awe_output.Host == "" {
			awe_output.Host = helper.job.ShockHost
		}
		awe_task.Outputs = append(awe_task.Outputs, awe_output)
	}

	//fmt.Println("=============================================== expected inputs")

	return
}

func cwl_step_2_awe_task(helper *Helper, step_id string) (err error) {
	logger.Debug(1, "cwl_step_2_awe_task, step_id: "+step_id)

	processed_ws := helper.processed_ws
	job := helper.job

	step, ok := (*helper.unprocessed_ws)[step_id]
	if !ok {
		err = fmt.Errorf("Step %s not found in unprocessed_ws", step_id)
		return
	}
	delete(*helper.unprocessed_ws, step_id) // will be added at the end of this function to processed_ws

	spew.Dump(step)
	fmt.Println("run: " + step.Run)
	cmdlinetool_str := ""
	if strings.HasPrefix(step.Run, "#") {
		// '#' it is a URI relative fragment identifier
		cmdlinetool_str = strings.TrimPrefix(step.Run, "#")

	}

	// TODO detect identifier URI (http://www.commonwl.org/v1.0/SchemaSalad.html#Identifier_resolution)
	// TODO: strings.Contains(step.Run, ":") or use split

	if cmdlinetool_str == "" {
		err = errors.New("Run string empty")
		return
	}

	CommandLineTools := helper.collection.CommandLineTools

	cmdlinetool, ok := CommandLineTools[cmdlinetool_str]

	if !ok {
		err = errors.New("CommandLineTool not found: " + cmdlinetool_str)
	}

	pos := len(*helper.processed_ws)
	logger.Debug(1, "pos: %d", pos)

	pos_str := strconv.Itoa(pos)
	logger.Debug(1, "pos_str: %s", pos_str)
	var awe_task *Task
	awe_task, err = NewTask(job, pos_str)
	if err != nil {
		err = fmt.Errorf("Task creation failed: %v", err)
		return
	}
	awe_task.Init(job)
	logger.Debug(1, "Task created: %s", awe_task.Id)

	(*helper.AWE_tasks)[job.Id] = awe_task
	awe_task.JobId = job.Id

	err = createAweTask(helper, cmdlinetool, step, awe_task)
	if err != nil {
		return
	}

	job.Tasks = append(job.Tasks, awe_task)

	(*processed_ws)[step.Id] = step
	logger.Debug(1, "LEAVING cwl_step_2_awe_task, step_id: "+step_id)
	return
}

func CWL2AWE(_user *user.User, files FormFiles, cwl_workflow *cwl.Workflow, collection *cwl.CWL_collection) (job *Job, err error) {

	//CommandLineTools := collection.CommandLineTools

	// check that all expected workflow inputs exist and that they have the correct type
	logger.Debug(1, "CWL2AWE starting")
	for _, input := range cwl_workflow.Inputs {
		// input is a cwl.InputParameter object

		spew.Dump(input)

		id := input.Id
		expected_types := input.Type
		obj_ref, errx := collection.Get(id)
		if errx != nil {

			fmt.Printf("-------Collection")
			spew.Dump(collection.All)

			fmt.Printf("-------")
			err = fmt.Errorf("value for workflow input \"%s\" not found", id)
			return
		}
		obj := *obj_ref
		obj_type := obj.GetClass()

		if !cwl.HasInputParameterType(expected_types, obj_type) {
			//if strings.ToLower(obj_type) != strings.ToLower(expected_types) {
			fmt.Printf("object found: ")
			spew.Dump(obj)
			err = fmt.Errorf("Input.type array does not accept \"%s\" (id=%s)", obj_type, id)
			return
		}

		_ = obj
		_ = obj_ref

	}
	//os.Exit(0)
	job = NewJob()
	logger.Debug(1, "Job created")

	found_ShockRequirement := false
	for _, r := range cwl_workflow.Requirements { // TODO put ShockRequirement in Hints
		switch r.GetClass() {
		case "ShockRequirement":
			sr, ok := r.(cwl.ShockRequirement)
			if !ok {
				err = fmt.Errorf("Could not assert ShockRequirement")
				return
			}

			job.ShockHost = sr.Host
			found_ShockRequirement = true

		}
	}

	if !found_ShockRequirement {
		err = fmt.Errorf("ShockRequirement has to be provided in the workflow object")
		return
	}
	logger.Debug(1, "Requirements checked")

	// Once, job has been created, set job owner and add owner to all ACL's
	job.Acl.SetOwner(_user.Uuid)
	job.Acl.Set(_user.Uuid, acl.Rights{"read": true, "write": true, "delete": true})

	// TODO first check that all resources are available: local files and remote links

	helper := Helper{}

	processed_ws := make(map[string]*cwl.WorkflowStep)
	unprocessed_ws := make(map[string]*cwl.WorkflowStep)
	awe_tasks := make(map[string]*Task)
	helper.processed_ws = &processed_ws
	helper.unprocessed_ws = &unprocessed_ws
	helper.collection = collection
	helper.job = job
	helper.AWE_tasks = &awe_tasks

	for _, step := range cwl_workflow.Steps {
		unprocessed_ws[step.Id] = &step
	}

	for {

		// find any step to start with
		step_id := ""
		for step_id, _ = range unprocessed_ws {
			break
		}

		err = cwl_step_2_awe_task(&helper, step_id)
		if err != nil {
			return
		}
		//job.Tasks = append(job.Tasks, awe_tasks)

		if len(unprocessed_ws) == 0 {
			break
		}

	}
	logger.Debug(1, "cwl_step_2_awe_task done")
	// loop until all steps have been converted

	//err = job.InitTasks()
	//if err != nil {
	//	err = fmt.Errorf("job.InitTasks() failed: %v", err)
	//	return
	//}
	//logger.Debug(1, "job.InitTasks done")

	_, err = job.Init()
	if err != nil {
		err = fmt.Errorf("job.Init() failed: %v", err)
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
