package core

import (
	"fmt"
	"strings"
)

type Task_Unique_Identifier struct {
	Id    string `bson:"taskid" json:"taskid"` // local identifier
	JobId string `bson:"jobid" json:"jobid"`
}

func New_Task_Unique_Identifier(old_style_id string) (t Task_Unique_Identifier, err error) {

	array := strings.Split(old_style_id, "_")

	if len(array) != 2 {
		err = fmt.Errorf("(New_Task_Unique_Identifier) Cannot parse task identifier: %s", old_style_id)
		return
	}

	t = Task_Unique_Identifier{JobId: array[0], Id: array[1]}

	return
}
