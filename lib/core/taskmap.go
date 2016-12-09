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

// func (qm *ServerMgr) copyTask(a *Task) (b *Task) {
// 	b = new(Task)
// 	*b = *a
// 	return
// }

// func (qm *ServerMgr) putTask(task *Task) {
// 	qm.taskLock.Lock()
// 	qm.taskMap[task.Id] = task
// 	qm.taskLock.Unlock()
// }

// func (qm *ServerMgr) updateTask(task *Task) {
// 	qm.taskLock.Lock()
// 	if _, ok := qm.taskMap[task.Id]; ok {
// 		qm.taskMap[task.Id] = task
// 	}
// 	qm.taskLock.Unlock()
// }

// func (qm *ServerMgr) getTask(id string) (*Task, bool) {
// 	qm.taskLock.RLock()
// 	defer qm.taskLock.RUnlock()
// 	if task, ok := qm.taskMap[id]; ok {
// 		copy := qm.copyTask(task)
// 		return copy, true
// 	}
// 	return nil, false
// }

// func (qm *ServerMgr) getAllTasks() (tasks []*Task) {
// 	qm.taskLock.RLock()
// 	defer qm.taskLock.RUnlock()
// 	for _, task := range qm.taskMap {
// 		copy := qm.copyTask(task)
// 		tasks = append(tasks, copy)
// 	}
// 	return
// }

// func (qm *ServerMgr) deleteTask(id string) {
// 	qm.taskLock.Lock()
// 	delete(qm.taskMap, id)
// 	qm.taskLock.Unlock()
// }

func (tm *TaskMap) SetState(id string, new_state string) (err error) {
	task, ok := tm.Get(id, true)
	if !ok {
		err = fmt.Errorf("(SetState) Task not found")
		return
	}
	task.State = new_state
	return
}

// func (qm *ServerMgr) taskStateChange(id string, new_state string) (err error) {
//
// 	qm.taskLock.Lock()
// 	defer qm.taskLock.Unlock()
// 	if task, ok := qm.taskMap[id]; ok {
// 		task.State = new_state
// 		return nil
// 	}
// 	return errors.New(fmt.Sprintf("task %s not found", id))
// }

// func (qm *ServerMgr) hasTask(id string) (has bool) {
// 	qm.taskLock.RLock()
// 	defer qm.taskLock.RUnlock()
// 	if _, ok := qm.taskMap[id]; ok {
// 		has = true
// 	} else {
// 		has = false
// 	}
// 	return
// }

// func (qm *ServerMgr) listTasks() (ids []string) {
// 	qm.taskLock.RLock()
// 	defer qm.taskLock.RUnlock()
// 	for id := range qm.taskMap {
// 		ids = append(ids, id)
// 	}
// 	return
// }
