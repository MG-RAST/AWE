package core

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/acl"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	"github.com/MG-RAST/AWE/lib/user"
	"github.com/davecgh/go-spew/spew"
	//aw_sequences"os"
	"strings"
)

type Helper struct {
	unprocessed_ws *map[string]*cwl.WorkflowStep
	processed_ws   *map[string]*cwl.WorkflowStep
	collection     *cwl.CWL_collection
	job            *Job
}


func parseSourceString(source string) (linked_step_name string, fieldname string, err error) {
	
	if !strings.HasPrefix(source, "#") {
		err = fmt.Errorf("source has to start with # (cwl_step %s)", id)
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

	collection := helper.collection
	//unprocessed_ws := helper.unprocessed_ws
	processed_ws := helper.processed_ws

	local_collection := cwl.NewCWL_collection()
	//workflowStepInputMap := make(map[string]cwl.WorkflowStepInput)

	linked_IO := make(map[string]*IO)

	spew.Dump(cwl_tool)
	spew.Dump(cwl_step)

	fmt.Println("=============================================== variables")
	// collect variables in local_collection
	for _, input := range cwl_step.In {
		// input is a cwl.WorkflowStepInput

		spew.Dump(input)
		id := input.Id

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
				linked_step_name, fieldname, err = parseSourceString(source)
				if err != nil {
					return
				}

				// find fieldname, may refer to other task, recurse in creation of that other task if necessary
				var linked_step WorkflowStepInput
				if linked_step_name != "" {
					linked_step, ok = (*processed_ws)[linked_step_name]
					if !ok {

						// recurse into depending step
						err = cwl_step_2_awe_task(helper, linked_step_name)
						if err != nil {
							err = fmt.Errorf("source empty (%s) (%s)", id, err.Error())
							return
						}

						linked_step, ok = (*processed_ws)[linked_step_name]
						if !ok {
							err = fmt.Errorf("linked_step %s not found after processing", linked_step_name)
							return
						}
					}
					// linked_step is defined now.
					_ = linked_step
					
					
					// search output of linked_step 
					
					linked_task_id := linked_output.AWE_task.Id
					
					linked_filename := ""
					for _, awe_output := range linked_step.AWE_task.Outputs {
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
					linked_IO[id] = new IO{Origin: linked_task_id, FileName: linked_filename, Name: id}
					
										
					// TODO STORE THIS und use below !
					
					
				} else {
					// use workflow input as input
					
					var obj * CWLObject
					obj, err = (*helper.collection).Get(fieldname)
					if err != nil {
						return
					}
					
					obj_local := *obj
					obj_local.Id := id
					local_collection.Add(obj_local)
					
					
				}
				

			} else {
				err = fmt.Errorf("MultipleInputFeatureRequirement not supported yet (id=%s)", id)
				return
			}
			
			
			
			var obj *CWL_object
			obj, err = input.GetObject()
			if err != nil {
				return
			}
			
			obj_local := *obj
			obj_local.Id := id
			local_collection.Add(obj_local) // add (global) object with local input name
		} else {

		
			var obj *CWLObject
			obj, err = input.GetObject() // from ValueFrom or Default fields
			if err != nil {
				return
			}
			obj_local := *obj
			obj_local.Id := id
			
			local_collection.Add(obj_local) // add object with local input name
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
		input_optional := strings.HasSuffix(expected_input.Type, "?")
		expected_input_type := strings.ToLower(strings.TrimSuffix(expected_input.Type, "?"))
		_ = input_optional

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
		//input_string := ""
		
		var obj *CWLObject 
		obj, err = local_collection.Get(id)
		if err != nil {
			
			var io_struct *IO
			io_struct, ok = linked_IO[id]
			if ! ok {
				err = fmt.Errorf("%s not found in local_collection or linked_IO", id)
				return
			}
			
			
			// TODO
			// io_struct.DataToken = ..

			awe_task.Inputs = append(awe_task.Inputs, io_struct)
			
			
		} else {
			
			obj_class := obj.GetClass()
			switch obj_class {
				
				
			case "File":
				
				
				var file_obj File
				file_obj, err = local_collection.GetFile(id)
				if err != nil {
					err = fmt.Errorf("%s not found in local_collection ", id)
					return
				}
				
				// TODO HERE HERE HERE HERE HERE HERE HERE HERE HERE HERE HERE HERE HERE
				input_string = input_file.Basename

				awe_input := NewIO()
				awe_input.FileName = input_file.Basename
				awe_input.Name = input_file.Id
				awe_input.Host = input_file.Host
				awe_input.Node = input_file.Node
				//awe_input.Url=input_file.
				awe_input.DataToken = input_file.Token

				awe_task.Inputs = append(awe_task.Inputs, awe_input)
			case "String":
				err = fmt.Error("not implemented yet (%s)", obj_class )
			case "Int":
				err = fmt.Error("not implemented yet (%s)", obj_class )
				
			default:
				
				err = fmt.Error("not implemented yet (%s)", obj_class )
				
				
				
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

		awe_task.Outputs = append(awe_task.Outputs, awe_output)
	}

	//fmt.Println("=============================================== expected inputs")

	return
}

func cwl_step_2_awe_task(helper *Helper, step_id string) (err error) {

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
	awe_task := NewTask(job, string(pos))
	step.AWE_task = awe_task
	awe_task.JobId = job.Id

	err = createAweTask(helper, &cmdlinetool, step, awe_task)
	if err != nil {
		return
	}

	job.Tasks = append(job.Tasks, awe_task)

	(*processed_ws)[step.Id] = step
	return
}

func CWL2AWE(_user *user.User, files FormFiles, cwl_workflow *cwl.Workflow, collection *cwl.CWL_collection) (job *Job, err error) {

	//CommandLineTools := collection.CommandLineTools

	// check that all expected workflow inputs exist and that they have the correct type

	for _, input := range cwl_workflow.Inputs {
		// input is a cwl.InputParameter object

		spew.Dump(input)

		id := input.Id
		expected_type := input.Type
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

		if strings.ToLower(obj_type) != strings.ToLower(expected_type) {
			fmt.Printf("object found: ")
			spew.Dump(obj)
			err = fmt.Errorf("Expected type \"%s\", but got \"%s\" (id=%s)", expected_type, obj_type, id)
			return
		}

		_ = obj
		_ = obj_ref

	}
	//os.Exit(0)
	job = NewJob()

	//job.initJob("0")

	// Once, job has been created, set job owner and add owner to all ACL's
	job.Acl.SetOwner(_user.Uuid)
	job.Acl.Set(_user.Uuid, acl.Rights{"read": true, "write": true, "delete": true})

	// TODO first check that all resources are available: local files and remote links

	helper := Helper{}

	processed_ws := make(map[string]*cwl.WorkflowStep)
	unprocessed_ws := make(map[string]*cwl.WorkflowStep)
	helper.processed_ws = &processed_ws
	helper.unprocessed_ws = &unprocessed_ws
	helper.collection = collection
	helper.job = job

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
	// loop until all steps have been converted

	err = job.Mkdir()
	if err != nil {
		err = errors.New("error creating job directory, error=" + err.Error())
		return
	}

	err = job.UpdateFile(files, "cwl") // TODO that may not make sense. Check if I should store AWE job.
	if err != nil {
		err = errors.New("error in UpdateFile, error=" + err.Error())
		return
	}

	err = job.Save()
	if err != nil {
		err = errors.New("error in job.Save(), error=" + err.Error())
		return
	}
	return

}
