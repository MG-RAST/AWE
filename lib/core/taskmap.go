package core

import (
	"fmt"
)

type TaskMap struct {
	RWMutex
	_map map[string]*Task
}

func NewTaskMap() (t *TaskMap) {
	t = &TaskMap{
		_map: make(map[string]*Task),
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

func (tm *TaskMap) Get(taskid string, lock bool) (task *Task, ok bool, err error) {
	if lock {
		read_lock, xerr := tm.RLockNamed("Len")
		if xerr != nil {
			err = xerr
			return
		}
		defer tm.RUnlockNamed(read_lock)
	}

	task, ok = tm._map[taskid]
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

func (tm *TaskMap) Delete(taskid string) (task *Task, ok bool) {
	tm.LockNamed("Delete")
	defer tm.Unlock()
	delete(tm._map, taskid) // TODO should get write lock on task first
	return
}

func (tm *TaskMap) Add(task *Task) {
	tm.LockNamed("Add")
	defer tm.Unlock()
	tm._map[task.Id] = task // TODO prevent overwriting
	return
}

// TODO remove ?
func (tm *TaskMap) SetState(id string, new_state string) (err error) {
	task, ok := tm.Get(id, true)
	if !ok {
		err = fmt.Errorf("(SetState) Task not found")
		return
	}
	task.SetState(new_state)
	return
}
