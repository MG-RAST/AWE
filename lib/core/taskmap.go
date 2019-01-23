package core

import (
	"fmt"

	"github.com/MG-RAST/AWE/lib/rwmutex"
)

type TaskMap struct {
	rwmutex.RWMutex
	_map map[Task_Unique_Identifier]*Task
}

func NewTaskMap() (t *TaskMap) {
	t = &TaskMap{
		_map: make(map[Task_Unique_Identifier]*Task),
	}
	t.RWMutex.Init("TaskMap")
	return t

}

//--------task accessor methods-------

func (tm *TaskMap) Len() (length int, err error) {
	read_lock, err := tm.RLockNamed("Len")
	if err != nil {
		return
	}
	defer tm.RUnlockNamed(read_lock)
	length = len(tm._map)
	return
}

func (tm *TaskMap) Get(taskid Task_Unique_Identifier, lock bool) (task *Task, ok bool, err error) {
	if lock {
		read_lock, xerr := tm.RLockNamed("Get")
		if xerr != nil {
			err = xerr
			return
		}
		defer tm.RUnlockNamed(read_lock)
	}

	task, ok = tm._map[taskid]
	return
}

func (tm *TaskMap) Has(taskid Task_Unique_Identifier, lock bool) (ok bool, err error) {
	if lock {
		read_lock, xerr := tm.RLockNamed("Get")
		if xerr != nil {
			err = xerr
			return
		}
		defer tm.RUnlockNamed(read_lock)
	}

	_, ok = tm._map[taskid]
	return
}

func (tm *TaskMap) GetTasks() (tasks []*Task, err error) {

	tasks = []*Task{}

	read_lock, err := tm.RLockNamed("GetTasks")
	if err != nil {
		return
	}
	defer tm.RUnlockNamed(read_lock)

	for _, task := range tm._map {
		tasks = append(tasks, task)
	}

	return
}

func (tm *TaskMap) Delete(taskid Task_Unique_Identifier) (task *Task, ok bool, err error) {
	err = tm.LockNamed("Delete")
	if err != nil {
		return
	}
	defer tm.Unlock()
	delete(tm._map, taskid) // TODO should get write lock on task first
	return
}

func (tm *TaskMap) Add(task *Task, caller string) (err error) {

	//if task.Comment != "" {

	//	err = fmt.Errorf("task.Comment not empty: %s", task.Comment)
	//	return
	//}

	if caller == "" {
		err = fmt.Errorf("caller empty")
		return
	}

	err = tm.LockNamed("Add")
	if err != nil {
		return
	}
	defer tm.Unlock()

	var id Task_Unique_Identifier
	id, err = task.GetId("TaskMap/Add")
	if err != nil {
		err = fmt.Errorf("(TaskMap/Add) task.GetId returned: %s", err.Error())
		return
	}
	id_str, _ := id.String()

	task_in_map, has_task := tm._map[id]
	if has_task && (task_in_map != task) {

		err = fmt.Errorf("(TaskMap/Add) task %s is already in TaskMap with a different pointer (caller: %s, comment: %s)", id_str, caller, task_in_map.Comment)
		return
	}

	var task_state string
	task_state, err = task.GetState()
	if err != nil {
		err = fmt.Errorf("(TaskMap/Add) task.GetState returned: %s", err.Error())
		return
	}

	if task_state == TASK_STAT_INIT {
		err = task.SetState(TASK_STAT_PENDING, true)
		if err != nil {
			err = fmt.Errorf("(TaskMap/Add) task.SetState returned: %s", err.Error())
			return
		}
	}

	task.Comment = "added by " + caller
	tm._map[id] = task
	return
}
