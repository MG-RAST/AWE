package core

import (
	"fmt"
)

type TaskMap struct {
	RWMutex
	Map map[string]*Task
}

func NewTaskMap() (t *TaskMap) {
	t = &TaskMap{
		Map: make(map[string]*Task),
	}
	t.RWMutex.Init("TaskMap")
	return t

}

//--------task accessor methods-------

func (tm *TaskMap) Len() int {
	read_lock := tm.RLockNamed("Len")
	defer tm.RUnlockNamed(read_lock)
	return len(tm.Map)
}

func (tm *TaskMap) Get(taskid string, lock bool) (task *Task, ok bool) {
	if lock {
		read_lock := tm.RLockNamed("Len")
		defer tm.RUnlockNamed(read_lock)
	}

	task, ok = tm.Map[taskid]
	return
}

func (tm *TaskMap) Delete(taskid string) (task *Task, ok bool) {
	tm.LockNamed("Delete")
	defer tm.Unlock()
	delete(tm.Map, taskid) // TODO should get write lock on task first
	return
}

func (tm *TaskMap) Add(task *Task) {
	tm.LockNamed("Add")
	defer tm.Unlock()
	tm.Map[task.Id] = task // TODO prevent overwriting
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
