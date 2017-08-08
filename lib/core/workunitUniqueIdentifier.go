package core

import (
	"fmt"
	"strconv"
	"strings"
)

type Workunit_Unique_Identifier struct {
	Rank   int    `bson:"rank" json:"rank"` // this is the local identifier
	TaskId string `bson:"taskid" json:"taskid"`
	JobId  string `bson:"jobid" json:"jobid"`
}

func (w Workunit_Unique_Identifier) String() string {
	return fmt.Sprintf("%s_%s_%d", w.JobId, w.TaskId, w.Rank)
}

func (w Workunit_Unique_Identifier) GetTask() Task_Unique_Identifier {
	return Task_Unique_Identifier{JobId: w.JobId, Id: w.TaskId}
}

func New_Workunit_Unique_Identifier(old_style_id string) (w Workunit_Unique_Identifier, err error) {

	array := strings.Split(old_style_id, "_")

	if len(array) != 3 {
		err = fmt.Errorf("Cannot parse workunit identifier: %s", old_style_id)
		return
	}

	rank, err := strconv.Atoi(array[2])
	if err != nil {
		return
	}

	if !IsValidUUID(array[0]) {
		err = fmt.Errorf("Cannot parse workunit identifier, job id is not valid uuid: %s", old_style_id)
		return
	}

	w = Workunit_Unique_Identifier{JobId: array[0], TaskId: array[1], Rank: rank}

	return
}
