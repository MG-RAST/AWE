package core

import (
	"fmt"
	"strings"
)

type Task_Unique_Identifier struct {
	TaskName  string `bson:"task_name" json:"task_name"` // example: #main/filter
	Ancestors string `bson:"ancestors" json:"ancestors"`
	JobId     string `bson:"jobid" json:"jobid"`
}

func New_Task_Unique_Identifier(jobid string, workflow string, taskname string) (t Task_Unique_Identifier) {
	return Task_Unique_Identifier{JobId: jobid, Ancestors: workflow, TaskName: taskname}
}

func New_Task_Unique_Identifier_FromString(old_style_id string) (t Task_Unique_Identifier, err error) {

	array := strings.SplitN(old_style_id, "_", 2) // splits only first underscore

	if len(array) != 2 {
		err = fmt.Errorf("(New_Task_Unique_Identifier_FromString) Cannot parse task identifier: %s", old_style_id)
		return
	}

	name_array := strings.Split(array[1], "/")
	name := ""
	workflow := ""
	s := len(name_array)
	if s == 0 {
		name = ""
	} else if s == 1 {
		name = name_array[0]
	} else {
		name = name_array[s-1]
		workflow = strings.Join(name_array[0:s-2], "/")
	}

	if !IsValidUUID(array[0]) {
		err = fmt.Errorf("Cannot parse workunit identifier, job id is not valid uuid: %s", old_style_id)
		return
	}

	t = New_Task_Unique_Identifier(array[0], workflow, name)

	return
}

func (taskid Task_Unique_Identifier) String() (s string) {

	jobId := taskid.JobId
	workflow := taskid.Ancestors
	name := taskid.TaskName

	if workflow != "" {
		s = fmt.Sprintf("%s_%s/%s", jobId, workflow, name)
	} else {
		s = fmt.Sprintf("%s_%s", jobId, name)
	}

	return
}
