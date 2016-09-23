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

func createAweTask(helper *Helper, cwl_tool *cwl.CommandLineTool, cwl_step *cwl.WorkflowStep, awe_task *Task) (err error) {

	// valueFrom is a StepInputExpressionRequirement
	// http://www.commonwl.org/v1.0/Workflow.html#StepInputExpressionRequirement

	collection := helper.collection
	//unprocessed_ws := helper.unprocessed_ws
	processed_ws := helper.processed_ws

	local_collection := cwl.NewCWL_collection()
	//workflowStepInputMap := make(map[string]cwl.WorkflowStepInput)

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

		if input.Source != nil {

			input_data_str := input.Source[0] /// TODO multiple vs single line

			fmt.Println("+++++ " + input_data_str)

			if strings.HasPrefix(input_data_str, "#") {
				input_obj_name := strings.TrimPrefix(input_data_str, "#")
				// find thingy in collection
				thingy, c_err := collection.Get(input_obj_name)
				if c_err != nil {
					fmt.Println("NOT FOUND input")
					err = c_err
					return
				} else {
					fmt.Println("FOUND input")
				}
				_ = thingy

				fmt.Println("=============================================== obj")
				spew.Dump(thingy)
			} else {
				err = fmt.Errorf("source without # prefix not implemented yet")
			}

		} else if string(input.ValueFrom) != "" {
			valueFrom_str := input.ValueFrom.Evaluate(helper.collection)

			input.ValueFrom = cwl.Expression(valueFrom_str)

		} else if input.Default != nil {
			fmt.Println("input.Default:", input.Default.String())

			switch input.Default.GetClass() {
			case "String":
				this_string := input.Default.(cwl.String)
				this_string.Value = (*helper.collection).Evaluate(this_string.Value)
				input.Default = this_string
			}

		} else {
			err = fmt.Errorf("no input (source, default or valueFrom) defined for %s", id)
			return
		}

		//workflowStepInputMap[id] = input
		local_collection.Add(input)
		fmt.Println("ADD local_collection.WorkflowStepInputs")
		spew.Dump(local_collection.WorkflowStepInputs)
	}
	//fmt.Println("workflowStepInputMap:")
	//spew.Dump(workflowStepInputMap)

	// copy cwl_tool to AWE via step
	command_line := ""
	fmt.Println("=============================================== expected inputs")
	for _, input := range cwl_tool.Inputs {
		// input is a cwl.CommandInputParameter

		id := input.Id
		_ = id
		input_optional := strings.HasSuffix(input.Type, "?")
		input_type := strings.ToLower(strings.TrimSuffix(input.Type, "?"))
		_ = input_optional

		// TODO lookup id in workflow step input

		fmt.Println("local_collection.WorkflowStepInputs")
		spew.Dump(local_collection.WorkflowStepInputs)
		//workflow_step_input, ok := workflowStepInputMap[id]
		workflow_step_input, xerr := local_collection.GetWorkflowStepInput(id)
		if xerr != nil {
			err = fmt.Errorf("%s not found in workflowStepInputMap", id)
			return
		}
		fmt.Println("workflow_step_input: ")
		spew.Dump(workflow_step_input)
		input_string := ""
		switch input_type {
		case "string":

			if len(workflow_step_input.Source) > 0 {
				if len(workflow_step_input.Source) == 1 {
					source := workflow_step_input.Source[0] // #fieldname or #stepname.fieldname

					if !strings.HasPrefix(source, "#") {
						err = fmt.Errorf("source has to start with # (%s)", id)
						return
					}

					source = strings.TrimPrefix(source, "#")

					source_array := strings.Split(source, ".")

					linked_step_name := ""
					fieldname := ""
					_ = fieldname
					switch len(source_array) {
					case 0:
						if !strings.HasPrefix(source, "#") {
							err = fmt.Errorf("source empty (%s)", id)
							return
						}
					case 1:
						fieldname = source_array[0]
					case 2:
						linked_step_name = source_array[0]
						fieldname = source_array[1]
					default:
						err = fmt.Errorf("source has too many fields (%s)", id)
						return
					}

					// TODO find fieldname, may refer to other task, recurse in creation of that other task if necessary

					if linked_step_name != "" {
						linked_step, ok := (*processed_ws)[linked_step_name]
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
					}

				} else {
					err = fmt.Errorf("MultipleInputFeatureRequirement not supported yet (id=%s)", id)
					return
				}

			} else {

				if workflow_step_input.Default != nil {
					input_cwl_string, ok := workflow_step_input.Default.(cwl.String)
					if !ok {
						err = fmt.Errorf("Could not parse string %s", id)
						return
					}
					input_string = collection.Evaluate(input_cwl_string.String())
					fmt.Printf("%s -> %s\n", input_cwl_string.String(), input_string)
				}
			}

		case "int":

			if workflow_step_input.Default != nil {
				input_cwl_int, ok := workflow_step_input.Default.(cwl.Int)
				if !ok {
					err = fmt.Errorf("Could not parse string %s", id)
					return
				}
				input_string = input_cwl_int.String()

			}
		case "file":

			for _, source := range workflow_step_input.Source {
				fieldname := ""

				if !strings.HasPrefix(source, "#") {
					err = fmt.Errorf("prefix # missing %s, %s", id, source)
					return
				}

				fieldname = strings.TrimPrefix(source, "#")

				input_file, errx := collection.GetFile(fieldname)
				if errx != nil {
					err = fmt.Errorf("%s not found in collection", fieldname)
					return
				}

				//fmt.Println("input_file.Location: ")
				//fmt.Println(input_file.Location)

				// if shock, extract filename if not specified

				//spew.Dump(input_file)
				//fmt.Println(".........")
				input_string = input_file.Basename

				awe_input := NewIO()
				awe_input.FileName = input_file.Basename
				awe_input.Name = input_file.Id
				awe_input.Host = input_file.Host
				awe_input.Node = input_file.Node
				//awe_input.Url=input_file.
				awe_input.DataToken = input_file.Token

				awe_task.Inputs = append(awe_task.Inputs, awe_input)
				//spew.Dump(source)
			}

		default:
			err = fmt.Errorf("input_type unknown: %s", input_type)
			return
		}

		input_binding := input.InputBinding

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
