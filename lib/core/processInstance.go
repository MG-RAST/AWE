package core

import "github.com/MG-RAST/AWE/lib/core/cwl"

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
	SetState(newState string, writeLock bool) (err error)
}

// ProcessInstanceBase _
type ProcessInstanceBase struct {
	WorkflowStep   *cwl.WorkflowStep `bson:"-" json:"-" mapstructure:"-"`
	WorkflowStepID string            `bson:"workflowstepid" json:"workflowstepid" mapstructure:"workflowstepid"`
	State          string            `bson:"state" json:"state" mapstructure:"state"`
	TaskType       string            `bson:"task_type" json:"task_type" mapstructure:"task_type"`
	//ParentWorkflow     string
	//ParentWorkflowStep string
	// or use *cwl.WorkflowStep in cache ?
}

// GetWorkflowStep _
func (p *ProcessInstanceBase) GetWorkflowStep() (ws *cwl.WorkflowStep) {
	ws = p.WorkflowStep
	return
}
