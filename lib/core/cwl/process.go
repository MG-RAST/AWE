package cwl

import (
	"fmt"
)

// needed for run in http://www.commonwl.org/v1.0/Workflow.html#WorkflowStep
// string | CommandLineTool | ExpressionTool | Workflow

type Process interface {
	CWL_object
	is_process()
}

type ProcessPointer struct {
	Id    string
	Value string
}

func (p *ProcessPointer) is_process() {}
func (p *ProcessPointer) GetClass() string {
	return "ProcessPointer"
}
func (p *ProcessPointer) GetId() string { return p.Id }
func (p *ProcessPointer) SetId(string)  {}

// returns CommandLineTool, ExpressionTool or Workflow
func NewProcess(original interface{}, collection *CWL_collection) (process Process, err error) {

	switch original.(type) {
	case string:
		original_str := original.(string)

		pp := ProcessPointer{Value: original_str}

		process = pp
		return

	default:
		err = fmt.Errorf("(NewProcess) type unknown")

	}

	return
}
