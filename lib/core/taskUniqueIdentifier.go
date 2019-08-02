package core

import (
	"fmt"
	"strings"
)

type Task_Unique_Identifier struct {
	TaskName string `bson:"task_name" json:"task_name" mapstructure:"task_name"` // example: #main/filter
	//Parent   string `bson:"parent" json:"parent" mapstructure:"parent"`
	JobId string `bson:"jobid" json:"jobid" mapstructure:"jobid"`
}

func New_Task_Unique_Identifier(jobid string, taskname string) (t Task_Unique_Identifier, err error) {

	if taskname == "" {
		err = fmt.Errorf("(New_Task_Unique_Identifier) taskname empty")
		return
	}

	fixed_taskname := strings.TrimSuffix(taskname, "/")

	t = Task_Unique_Identifier{JobId: jobid, TaskName: fixed_taskname}

	return
}

func New_Task_Unique_Identifier_FromString(old_style_id string) (t Task_Unique_Identifier, err error) {

	job_id := old_style_id[:36]
	tail := old_style_id[37:]
	//array := strings.SplitN(old_style_id, "_", 2) // splits only first underscore

	if tail == "" {
		err = fmt.Errorf("(New_Task_Unique_Identifier_FromString) taskname is empty: %s", old_style_id)
		return
	}

	//if len(array) != 2 {
	//	err = fmt.Errorf("(New_Task_Unique_Identifier_FromString) Cannot parse task identifier: %s", old_style_id)
	//	return
	//}

	//job_id := array[0]
	if !IsValidUUID(job_id) {
		err = fmt.Errorf("(New_Task_Unique_Identifier_FromString) Cannot parse workunit identifier, job id is not valid uuid: %s", old_style_id)
		return
	}

	if IsValidUUID(tail) {
		err = fmt.Errorf("(New_Task_Unique_Identifier_FromString) Cannot parse workunit identifier, second item should not be uuid: %s", old_style_id)
		return
	}

	//task_string := array[1]

	//workflow := ""
	//name := ""

	// if strings.HasPrefix(tail, "#") {
	// 	// CWL job

	// 	//cwl_array := strings.Split(tail, "#")
	// 	// := len(cwl_array) // s has length 2 at least

	// 	// clean-up
	// 	for i := 0; i < len(cwl_array)-1; i++ {
	// 		cwl_array[i] = strings.TrimSuffix(cwl_array[i], "/")
	// 	}

	// 	workflow = strings.Join(cwl_array[0:s-2], "#")

	// 	name = "#" + strings.TrimSuffix(cwl_array[s-1], "/") // last element

	// } else {
	// 	// old-style AWE
	// 	name = task_string
	// }

	t, err = New_Task_Unique_Identifier(job_id, tail)
	if err != nil {
		err = fmt.Errorf("(New_Task_Unique_Identifier_FromString) New_Task_Unique_Identifier returned: %s", err.Error())
		return
	}

	//fmt.Printf("(New_Task_Unique_Identifier_FromString) new id: (%s) %s %s %s\n", old_style_id, job_id, workflow, name)

	return
}

func (taskid Task_Unique_Identifier) String() (s string, err error) {

	jobId := taskid.JobId
	//parent := taskid.Parent
	name := taskid.TaskName

	if name == "" {
		err = fmt.Errorf("(Task_Unique_Identifier/String) TaskName is empty!")
		return
	}

	if jobId == "" {
		err = fmt.Errorf("(Task_Unique_Identifier/String) JobId is empty!")
		return
	}

	s = fmt.Sprintf("%s_%s", jobId, name)

	return
}

func (taskid Task_Unique_Identifier) GetIDStr() (result string) {
	result, _ = taskid.String()
	return
}
