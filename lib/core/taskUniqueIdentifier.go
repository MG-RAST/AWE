package core

import (
	"fmt"
	"strings"
)

type Task_Unique_Identifier struct {
	TaskName  string   `bson:"task_name" json:"task_name"` // example: #main/filter
	Ancestors []string `bson:"ancestors" json:"ancestors"`
	JobId     string   `bson:"jobid" json:"jobid"`
}

func New_Task_Unique_Identifier(jobid string, workflow []string, taskname string) (t Task_Unique_Identifier) {
	return Task_Unique_Identifier{JobId: jobid, Ancestors: workflow, TaskName: taskname}
}

func New_Task_Unique_Identifier_FromString(old_style_id string) (t Task_Unique_Identifier, err error) {

	array := strings.SplitN(old_style_id, "_", 2) // splits only first underscore

	if len(array) != 2 {
		err = fmt.Errorf("(New_Task_Unique_Identifier_FromString) Cannot parse task identifier: %s", old_style_id)
		return
	}

	job_id := array[0]
	if !IsValidUUID(job_id) {
		err = fmt.Errorf("Cannot parse workunit identifier, job id is not valid uuid: %s", old_style_id)
		return
	}

	task_string := array[1]

	workflow := []string{}
	name := ""

	if strings.HasPrefix(task_string, "#") {
		// CWL job

		cwl_array := strings.Split(task_string, "#")

		for i := 0; i < len(cwl_array)-1; i++ {
			workflow = append(workflow, cwl_array[i])
		}

		s := len(cwl_array) // s has length 2 at least

		name = "#" + cwl_array[s-1] // last element
		workflow = strings.Join(cwl_array[0:s-2], "#")

	} else {
		// old-style AWE
		name = task_string
	}

	t = New_Task_Unique_Identifier(job_id, workflow, name)

	return
}

func (taskid Task_Unique_Identifier) String() (s string) {

	jobId := taskid.JobId
	workflow := strings.Join(taskid.Ancestors, "")
	name := taskid.TaskName

	if workflow != "" {
		s = fmt.Sprintf("%s_%s/%s", jobId, workflow, name)
	} else {
		s = fmt.Sprintf("%s_%s", jobId, name)
	}

	return
}
