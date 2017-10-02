package core

import (
	"fmt"
	"strings"
)

type Task_Unique_Identifier struct {
	TaskName string `bson:"task_name" json:"task_name"` // local identifier
	Workflow string `bson:"workflow" json:"workflow"`
	JobId    string `bson:"jobid" json:"jobid"`
}

func New_Task_Unique_Identifier(old_style_id string) (t Task_Unique_Identifier, err error) {

	array := strings.SplitN(old_style_id, "_", 2) // splits only first underscore

	if len(array) != 2 {
		err = fmt.Errorf("(New_Task_Unique_Identifier) Cannot parse task identifier: %s", old_style_id)
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

	t = Task_Unique_Identifier{JobId: array[0], Workflow: workflow, Name: array[1]}

	return
}
