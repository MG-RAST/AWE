package core

import ()

type CWL_workunit struct {
	Job_input          *cwl.Job_document
	Job_input_filename string
	CWL_tool           *cwl.CommandLineTool
	CWL_tool_filename  string
	Tool_results       *cwl.Job_document
	OutputsExpected    *cwl.WorkflowStepOutput // this is the subset of outputs that are needed by the workflow
}

func NewCWL_workunit() *CWL_workunit {
	return &CWL_workunit{
		Job_input:       nil,
		CWL_tool:        nil,
		Tool_results:    nil,
		OutputsExpected: nil,
	}

}

func NewCWL_workunit_from_interface(native interface{}) (workunit *CWL_workunit, err error) {

	workunit = NewCWL_workunit()

	switch native.(type) {

	case map[string]interface{}:

	default:
		err = fmt.Errorf("(NewCWL_workunit_from_interface) wrong type, map expected")
		return

	}

	return

}
