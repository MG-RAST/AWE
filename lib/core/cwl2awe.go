package core

import (
	"errors"
	"github.com/MG-RAST/AWE/lib/acl"
	"github.com/MG-RAST/AWE/lib/core/cwl" // namespace would conflict
	"github.com/MG-RAST/AWE/lib/user"
)

func CWL2AWE(_user *user.User, cwl_workflow cwl.Workflow, CommandLineTools []cwl.CommandLineTool) (err error, job *Job) {

	// compare with core.CreateJobUpload

	job = NewJob("")

	//job.initJob("0")

	// Once, job has been created, set job owner and add owner to all ACL's
	job.Acl.SetOwner(_user.Uuid)
	job.Acl.Set(_user.Uuid, acl.Rights{"read": true, "write": true, "delete": true})

	err = job.Mkdir()
	if err != nil {
		err = errors.New("error creating job directory, error=" + err.Error())
		return
	}

	return
}
