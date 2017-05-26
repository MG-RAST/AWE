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

// returns CommandLineTool, ExpressionTool or Workflow
func NewProcess(original interface{}, collection *CWL_collection) (process Process, err error) {

	switch original.(type) {
	case string:
		original_str := original.(string)

		something_ptr, xerr := collection.Get(original_str)

		if xerr != nil {
			err = fmt.Errorf("%s not found", original_str, xerr.Error())
			return
		}

		something := *something_ptr

		var ok bool
		process, ok = something.(Process)
		if !ok {
			err = fmt.Errorf("%s does not seem to be a process: %s", original_str)
			return
		}

	default:
		err = fmt.Errorf("(NewProcess) type unknown")

	}

	return
}
