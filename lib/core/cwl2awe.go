package core

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/acl"
	"github.com/MG-RAST/AWE/lib/core/cwl" // namespace would conflict
	"github.com/MG-RAST/AWE/lib/user"
	"github.com/davecgh/go-spew/spew"
	"strings"
)

func CWL2AWE(_user *user.User, files FormFiles, cwl_workflow cwl.Workflow, CommandLineTools map[string]cwl.CommandLineTool) (err error, job *Job) {

	// collect workflow inputs

	job = NewJob("")

	//job.initJob("0")

	// Once, job has been created, set job owner and add owner to all ACL's
	job.Acl.SetOwner(_user.Uuid)
	job.Acl.Set(_user.Uuid, acl.Rights{"read": true, "write": true, "delete": true})

	// TODO first check that all resources are available: local files and remote links

	for _, step := range cwl_workflow.Steps {

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
		//awe_task = NewTask

	} // edn for

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
