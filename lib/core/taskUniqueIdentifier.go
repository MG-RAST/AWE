package core

import (
	"fmt"
	"strings"
)

type Task_Unique_Identifier struct {
	TaskName string `bson:"task_name" json:"task_name" mapstructure:"task_name"` // example: #main/filter
	Parent   string `bson:"parent" json:"parent" mapstructure:"parent"`
	JobId    string `bson:"jobid" json:"jobid" mapstructure:"jobid"`
}

func New_Task_Unique_Identifier(jobid string, parent string, taskname string) (t Task_Unique_Identifier, err error) {

	if taskname == "" {
		err = fmt.Errorf("(New_Task_Unique_Identifier) taskname empty")
	}

	fixed_taskname := strings.TrimSuffix(taskname, "/")

	t = Task_Unique_Identifier{JobId: jobid, Parent: parent, TaskName: fixed_taskname}

	return
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

	workflow := ""
	name := ""

	if strings.HasPrefix(task_string, "#") {
		// CWL job

		cwl_array := strings.Split(task_string, "#")
		s := len(cwl_array) // s has length 2 at least

		// clean-up
		for i := 0; i < len(cwl_array)-1; i++ {
			cwl_array[i] = strings.TrimSuffix(cwl_array[i], "/")
		}

		workflow = strings.Join(cwl_array[0:s-2], "#")

		name = "#" + strings.TrimSuffix(cwl_array[s-1], "/") // last element

	} else {
		// old-style AWE
		name = task_string
	}

	t, err = New_Task_Unique_Identifier(job_id, workflow, name)
	if err != nil {
		return
	}

	//fmt.Printf("(New_Task_Unique_Identifier_FromString) new id: (%s) %s %s %s\n", old_style_id, job_id, workflow, name)

	return
}

func (taskid Task_Unique_Identifier) String() (s string) {

	jobId := taskid.JobId
	parent := taskid.Parent
	name := taskid.TaskName

	if name == "" {
		name = "ERROR"
	}

	if parent != "" {
		s = fmt.Sprintf("%s_%s/%s", jobId, parent, name)
	} else {
		s = fmt.Sprintf("%s_%s", jobId, name)
	}

	return
}
