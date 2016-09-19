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

func createAweTask(collection *cwl.CWL_collection, cwl_tool *cwl.CommandLineTool, cwl_step *cwl.WorkflowStep, awe_task *Task) (err error) {

	// valueFrom is a StepInputExpressionRequirement
	// http://www.commonwl.org/v1.0/Workflow.html#StepInputExpressionRequirement

	workflowStepInputMap := make(map[string]cwl.WorkflowStepInput)

	spew.Dump(cwl_tool)
	spew.Dump(cwl_step)

	fmt.Println("=============================================== variables")
	for _, input := range cwl_step.In {
		// input is a cwl.WorkflowStepInput

		spew.Dump(input)
		id := input.Id

		if id == "" {
			err = fmt.Errorf("id is empty")
			return
		}

		fmt.Println("//////////////// step input: " + id)
		_ = id
		if input.Source != nil {

			input_data_str := input.Source[0] /// TODO multiple vs single line

			fmt.Println("+++++ " + input_data_str)

			if strings.HasPrefix(input_data_str, "#") {
				input_obj_name := strings.TrimPrefix(input_data_str, "#")
				// find thingy in collection
				thingy, c_err := collection.Get(input_obj_name)
				if c_err != nil {
					fmt.Println("NOT FOUND input")
					return c_err
				} else {
					fmt.Println("FOUND input")
				}
				_ = thingy

				fmt.Println("=============================================== obj")
				spew.Dump(thingy)
			} else {
				err = fmt.Errorf("source without # prefix not implemented yet")
			}

		} else if input.Default != nil {
			fmt.Println("input.Default:", input.Default.String())
		} else {
			err = fmt.Errorf("no input (source or default) defined for %s", id)
			return
		}
		workflowStepInputMap[id] = input

	}
	fmt.Println("workflowStepInputMap:")
	spew.Dump(workflowStepInputMap)

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

		workflow_step_input, ok := workflowStepInputMap[id]
		if !ok {
			err = fmt.Errorf("%s not found in workflowStepInputMap", id)
			return
		}
		fmt.Println("workflow_step_input: ")
		spew.Dump(workflow_step_input)
		input_string := ""
		switch input_type {
		case "string":

			if workflow_step_input.Default != nil {
				input_cwl_string, ok := workflow_step_input.Default.(cwl.String)
				if !ok {
					err = fmt.Errorf("Could not parse string %s", id)
					return
				}
				input_string = input_cwl_string.String()

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
					err = fmt.Errorf("%s not found in workflowStepInputMap", fieldname)
					return
				}

				fmt.Println("input_file: ")
				fmt.Println(input_file.Path)
				spew.Dump(input_file)
				fmt.Println(".........")

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

	//fmt.Println("=============================================== expected inputs")

	return
}

func CWL2AWE(_user *user.User, files FormFiles, cwl_workflow *cwl.Workflow, collection *cwl.CWL_collection) (err error, job *Job) {

	CommandLineTools := collection.CommandLineTools

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

	for pos, step := range cwl_workflow.Steps {

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

		cmdlinetool, ok := CommandLineTools[cmdlinetool_str]

		if !ok {
			err = errors.New("CommandLineTool not found: " + cmdlinetool_str)
		}

		awe_task := NewTask(job, string(pos))

		awe_task.JobId = job.Id

		err = createAweTask(collection, &cmdlinetool, &step, awe_task)
		if err != nil {
			return
		}

		_ = awe_task

		job.Tasks = append(job.Tasks, awe_task)

	} // end for

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
