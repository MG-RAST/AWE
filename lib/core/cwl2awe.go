package core

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/acl"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	"github.com/MG-RAST/AWE/lib/user"
	"github.com/davecgh/go-spew/spew"
	"strings"
)

func createAweCmd(cwl_tool *cwl.CommandLineTool, cwl_step *cwl.WorkflowStep) (err error) {

	// valueFrom is a StepInputExpressionRequirement
	// http://www.commonwl.org/v1.0/Workflow.html#StepInputExpressionRequirement

	variable_strings := make(map[string][]string)

	spew.Dump(cwl_tool)
	spew.Dump(cwl_step)

	fmt.Println("=============================================== variables")
	for _, input := range cwl_step.In {
		// input is a cwl.WorkflowStepInput

		spew.Dump(input)
		id := input.Id
		input_data := []string{}

		if input.Source != nil {
			input_data = input.Source
			fmt.Println("+++++ " + input_data[0])

		} else {
			if input.Default != nil {
				input_data = input.Default
			} else {
				//if !input_optional {
				//	fmt.Println("=+++++++++++ " + id)
				//	err = errors.New("Input " + id + " not defined")
				//	spew.Dump(input)
				//	return
				//}
			}
		}

		variable_strings[id] = input_data
		spew.Dump(variable_strings)
	}

	fmt.Println("=============================================== expected inputs")
	for _, input := range cwl_tool.Inputs {
		// input is a cwl.CommandInputParameter

		id := input.Id
		_ = id
		input_optional := strings.HasSuffix(input.Type, "?")
		input_type := strings.TrimSuffix(input.Type, "?")

		_ = input_optional
		_ = input_type

		// Id:  #fieldname or #stepname.fieldname
		//switch input.Source {
		//case

		//}
	}

	//fmt.Println("=============================================== expected inputs")

	return
}

func CWL2AWE(_user *user.User, files FormFiles, cwl_workflow *cwl.Workflow, cwl_container *cwl.CWL_collection) (err error, job *Job) {

	CommandLineTools := cwl_container.CommandLineTools

	// collect workflow inputs

	job = NewJob("")

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

		_ = cmdlinetool

		err = createAweCmd(&cmdlinetool, &step)
		if err != nil {
			return
		}

		awe_task := NewTask(job, string(pos))

		awe_task.JobId = job.Id
		//awe_task. step.Id

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
