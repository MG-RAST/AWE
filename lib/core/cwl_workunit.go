package core

import (
	"github.com/MG-RAST/AWE/lib/core/cwl"
	//cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	//"github.com/davecgh/go-spew/spew"
	"fmt"
)

type CWL_workunit struct {
	Job_input          *cwl.Job_document       `bson:"job_input,omitempty" json:"job_input,omitempty"`
	Job_input_filename string                  `bson:"job_input_filename,omitempty" json:"job_input_filename,omitempty"`
	CWL_tool           *cwl.CommandLineTool    `bson:"cwl_tool,omitempty" json:"cwl_tool,omitempty"`
	CWL_tool_filename  string                  `bson:"cwl_tool_filename,omitempty" json:"cwl_tool_filename,omitempty"`
	Tool_results       *cwl.Job_document       `bson:"tool_results,omitempty" json:"tool_results,omitempty"`
	OutputsExpected    *cwl.WorkflowStepOutput `bson:"outputs_expected,omitempty" json:"outputs_expected,omitempty"` // this is the subset of outputs that are needed by the workflow
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

	workunit = &CWL_workunit{}

	switch native.(type) {

	case map[string]interface{}:

		native_map, ok := native.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewCWL_workunit_from_interface) type error")
			return
		}

		job_input_generic, has_job_input_generic := native_map["job_input"]
		if has_job_input_generic {

			job_input, xerr := cwl.NewJob_document(job_input_generic)
			if xerr != nil {
				err = fmt.Errorf("(NewCWL_workunit_from_interface) NewJob_document failed: %s", xerr.Error())
				return
			}
			workunit.Job_input = job_input

		}

		workunit.Job_input_filename, _ = native_map["Job_input_filename"].(string)
		workunit.CWL_tool_filename, _ = native_map["CWL_tool_filename"].(string)

		outputs_expected_generic, has_outputs_expected := native_map["outputs_expected"]
		if has_outputs_expected {

			outputs_expected, xerr := cwl.NewWorkflowStepOutput(outputs_expected_generic)
			if xerr != nil {
				err = fmt.Errorf("(NewCWL_workunit_from_interface) NewWorkflowStepOutput failed: %s", xerr.Error())
				return
			}

			workunit.OutputsExpected = outputs_expected
		}

		cwl_tool_generic, has_cwl_tool_generic := native_map["cwl_tool"]
		if has_cwl_tool_generic {

			cwl_tool, xerr := cwl.NewCommandLineTool(cwl_tool_generic)
			if xerr != nil {
				err = fmt.Errorf("(NewCWL_workunit_from_interface) NewCommandLineTool failed: %s", xerr.Error())
				return
			}
			workunit.CWL_tool = cwl_tool

		}

	default:
		err = fmt.Errorf("(NewCWL_workunit_from_interface) wrong type, map expected")
		return

	}

	return

}
