package core

import (
	"fmt"

	"github.com/MG-RAST/AWE/lib/core/cwl"
	"github.com/MG-RAST/AWE/lib/rwmutex"
)

// should replace WorkflowInstance states and tasks states
const (
	// ProcessStatInit        : initial state on creation of a task
	ProcessStatInit = "init"
	// ProcessStatPending     : a process that wants to be enqueued (but dependent process are not complete)
	ProcessStatPending = "pending" //
	// ProcessStatReady       : a process ready to be enqueued (all dependent process are complete , but sub-processes habe not yet been created)
	ProcessStatReady = "ready"
	// ProcessStatQueued      : a process for which sub-processes have been created/queued
	ProcessStatQueued = "queued"
	// ProcessStatInprogress  : a first workunit has been checkout (this does not guarantee a workunit is running right now)
	ProcessStatInprogress = "in-progress"
	// ProcessStatSuspend _
	ProcessStatSuspend = "suspend"
	// ProcessStatFailedPermanent on exit code 42
	ProcessStatFailedPermanent = "failed-permanent"
	// ProcessStatCompleted _
	ProcessStatCompleted = "completed"
)

const (
	// ProcessTypeUnkown _
	ProcessTypeUnkown = ""
	// ProcessTypeScatter -
	ProcessTypeScatter = "scatter"
	//TASK_TYPE_WORKFLOW = "workflow"
	// ProcessTypeNormal _
	ProcessTypeNormal = "normal"
)

// ProcessInstance _
// combines WorkflowInstance nad Task into one conceptual process type
type ProcessInstance interface {
	IsProcessInstance()
	GetWorkflowStep() *cwl.WorkflowStep
	GetIDStr() (result string)
	SetState(newState string, writeLock bool, caller string) (err error)
	SetProcessType(t string, lock bool) (err error)
	SetWorkflowStepID(pIf ProcessInstance, stepID string, lock bool) (err error)
	SetWorkflowStep(ws *cwl.WorkflowStep, lock bool) (err error)
}

// ProcessInstanceBase _
type ProcessInstanceBase struct {
	rwmutex.RWMutex `bson:"-" json:"-" mapstructure:"-"`
	WorkflowStep    *cwl.WorkflowStep `bson:"-" json:"-" mapstructure:"-"`
	WorkflowStepID  string            `bson:"workflowstepid" json:"workflowstepid" mapstructure:"workflowstepid"`
	State           string            `bson:"state" json:"state" mapstructure:"state"`
	ProcessType     string            `bson:"processtype" json:"processtype" mapstructure:"processtype"`
	ScatterChildren []string          `bson:"scatterChildren" json:"scatterChildren" mapstructure:"scatterChildren"` // use simple TaskName/WorkflowInstance id  , list of all children in a subworkflow task
	//ParentWorkflow     string
	//ParentWorkflowStep string
	// or use *cwl.WorkflowStep in cache ?
}

// GetWorkflowStep _
func (p *ProcessInstanceBase) GetWorkflowStep() (ws *cwl.WorkflowStep) {
	ws = p.WorkflowStep
	return
}

// GetProcessType _
func (p *ProcessInstanceBase) GetProcessType() (prType string, err error) {
	lock, err := p.RLockNamed("GetProcessType")
	if err != nil {
		return
	}
	defer p.RUnlockNamed(lock)
	prType = p.ProcessType
	return
}

// SetWorkflowStepID _
func (p *ProcessInstanceBase) SetWorkflowStepID(pIf ProcessInstance, stepID string, lock bool) (err error) {
	if lock {
		err = p.LockNamed("SetWorkflowStepID")
		if err != nil {
			return
		}
		defer p.Unlock()
	}

	switch pIf.(type) {

	case *WorkflowInstance:
		wi := pIf.(*WorkflowInstance)
		err = dbUpdateWorkflowInstancesFieldString(wi.ID, "workflowstepid", stepID)
		if err != nil {
			err = fmt.Errorf("(WorkflowInstance/SetWorkflowStepID) (wi.ID: %s) dbUpdateJobWorkflowInstancesFieldString returned: %s", wi.ID, err.Error())
			return
		}
		wi.WorkflowStepID = stepID
	case *Task:
		task := pIf.(*Task)
		err = dbUpdateTaskString(task.WorkflowInstanceUUID, task.ID, "workflowstepid", stepID)
		if err != nil {
			err = fmt.Errorf("(task/SetWorkflowStepID) dbUpdateTaskTime returned: %s", err.Error())
			return
		}
		task.WorkflowStepID = stepID
	}

	return
}

// SetWorkflowStep _
func (p *ProcessInstanceBase) SetWorkflowStep(ws *cwl.WorkflowStep, lock bool) (err error) {
	if lock {
		err = p.LockNamed("SetWorkflowStepID")
		if err != nil {
			return
		}
		defer p.Unlock()
	}

	p.WorkflowStep = ws

	return
}
