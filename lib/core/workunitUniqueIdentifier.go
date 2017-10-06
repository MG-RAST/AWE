package core

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
	"strconv"
	"strings"
)

type Workunit_Unique_Identifier struct {
	Task_Unique_Identifier `bson:",inline" json:",inline" mapstructure:",squash"` // TaskName, Workflow, JobId
	Rank                   int                                                    `bson:"rank" json:"rank" mapstructure:"rank"` // this is the local identifier

}

func New_Workunit_Unique_Identifier(task Task_Unique_Identifier, rank int) (wui Workunit_Unique_Identifier) {

	wui = Workunit_Unique_Identifier{}
	wui.Task_Unique_Identifier = task
	wui.Rank = rank

	return
}

func (w Workunit_Unique_Identifier) String() string {

	task_string := w.Task_Unique_Identifier.String()

	return fmt.Sprintf("%s_%d", task_string, w.Rank)
}

func (w Workunit_Unique_Identifier) GetTask() Task_Unique_Identifier {
	return w.Task_Unique_Identifier
}

func New_Workunit_Unique_Identifier_from_interface(original interface{}) (wui Workunit_Unique_Identifier, err error) {

	wui = Workunit_Unique_Identifier{}

	err = mapstructure.Decode(original, &wui)
	if err != nil {
		err = fmt.Errorf("(New_Workunit_Unique_Identifier_from_interface) mapstructure returs: %s", err.Error())
		return
	}

	return
}

func New_Workunit_Unique_Identifier_FromString(old_style_id string) (w Workunit_Unique_Identifier, err error) {

	array := strings.Split(old_style_id, "_")

	//if len(array) != 3 {
	//	err = fmt.Errorf("Cannot parse workunit identifier: %s", old_style_id)
	//	return
	//}

	a_len := len(array)

	rank_string := array[a_len-1]
	rank, err := strconv.Atoi(rank_string)
	if err != nil {
		return
	}

	prefix := ""
	if len(array) > 1 {
		prefix = strings.Join(array[0:a_len-1], "_")
	}

	var t Task_Unique_Identifier
	t, err = New_Task_Unique_Identifier_FromString(prefix)
	if err != nil {
		err = fmt.Errorf("(New_Workunit_Unique_Identifier_FromString) New_Task_Unique_Identifier_FromString returns: %s (prefix=%s, old_style_id=%s)", err.Error(), prefix, old_style_id)
		return
	}

	w = New_Workunit_Unique_Identifier(t, rank)

	return
}
