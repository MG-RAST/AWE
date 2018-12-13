package core

import "fmt"

type WorkflowInstanceMap struct {
	RWMutex
	_map map[string]*WorkflowInstance
}

func NewWorkflowInstancesMap() (wim *WorkflowInstanceMap) {

	wim = &WorkflowInstanceMap{}
	wim.RWMutex.Init("WorkflowInstancesMap")
	wim._map = make(map[string]*WorkflowInstance)
	return
}

func (wim *WorkflowInstanceMap) GetWorkflowInstances() (wis []*WorkflowInstance, err error) {

	wis = []*WorkflowInstance{}

	read_lock, err := wim.RLockNamed("GetWorkflowInstances")
	if err != nil {
		return
	}
	defer wim.RUnlockNamed(read_lock)

	for i, _ := range wim._map {
		wi := wim._map[i]
		wis = append(wis, wi)
	}

	return
}

func (wim *WorkflowInstanceMap) Add(workflow_instance *WorkflowInstance) (err error) {
	err = wim.LockNamed("WorkflowInstanceMap/Add")
	if err != nil {
		return
	}
	defer wim.Unlock()

	id, _ := workflow_instance.GetId(false)

	_, ok := wim._map[id]
	if ok {
		err = fmt.Errorf("(WorkflowInstanceMap/Add) workflow_instance %s already in map", id)
		return
	}

	wim._map[id] = workflow_instance
	return
}

func (wim *WorkflowInstanceMap) Get(id string) (workflow_instance *WorkflowInstance, ok bool, err error) {
	rlock, err := wim.RLockNamed("WorkflowInstanceMap/Get")
	if err != nil {
		return
	}
	defer wim.RUnlockNamed(rlock)
	workflow_instance, ok = wim._map[id]
	return
}
