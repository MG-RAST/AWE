package core

import "github.com/MG-RAST/AWE/lib/core/cwl"

// ProcessInstance _
// combines WorkflowInstance nad Task into one conceptual process type
type ProcessInstance interface {
	IsProcessInstance()
	GetWorkflowStep() *cwl.WorkflowStep
	GetIDStr() (result string)
}

// ProcessInstanceBase _
type ProcessInstanceBase struct {
	WorkflowStep   *cwl.WorkflowStep `bson:"-" json:"-" mapstructure:"-"`
	WorkflowStepID string            `bson:"workflowstepid" json:"workflowstepid" mapstructure:"workflowstepid"`
	//ParentWorkflow     string
	//ParentWorkflowStep string
	// or use *cwl.WorkflowStep in cache ?
}

// GetWorkflowStep _
func (p *ProcessInstanceBase) GetWorkflowStep() (ws *cwl.WorkflowStep) {
	ws = p.WorkflowStep
	return
}
