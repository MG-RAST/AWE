package core

type WorkflowInstanceMap struct {
	RWMutex
	Map map[string]*WorkflowInstance
}

func NewWorkflowInstancesMap() (wim *WorkflowInstanceMap) {

	wim = &WorkflowInstanceMap{}
	wim.RWMutex.Init("WorkflowInstancesMap")
	wim.Map = make(map[string]*WorkflowInstance)
	return
}

func (wim *WorkflowInstanceMap) Add(workflow_instance *WorkflowInstance) (err error) {
	err = wim.LockNamed("WorkflowInstanceMap/Add")
	if err != nil {
		return
	}
	defer wim.Unlock()

	wim.Map[workflow_instance._Id] = workflow_instance
	return
}

func (wim *WorkflowInstanceMap) Get(id string) (workflow_instance *WorkflowInstance, ok bool, err error) {
	rlock, err := wim.RLockNamed("WorkflowInstanceMap/Get")
	if err != nil {
		return
	}
	defer wim.RUnlockNamed(rlock)
	workflow_instance, ok = wim.Map[id]
	return
}
