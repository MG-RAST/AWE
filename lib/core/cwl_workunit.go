package core

import (
	"github.com/MG-RAST/AWE/lib/core/cwl"
	//cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	//"github.com/davecgh/go-spew/spew"
	"fmt"
)

// CWLWorkunit _
type CWLWorkunit struct {
	JobInput         *cwl.Job_document `bson:"job_input,omitempty" json:"job_input,omitempty" mapstructure:"job_input,omitempty"`
	JobInputFilename string            `bson:"job_input_filename,omitempty" json:"job_input_filename,omitempty" mapstructure:"job_input_filename,omitempty"`
	//CWL_tool           *cwl.CommandLineTool      `bson:"cwl_tool,omitempty" json:"cwl_tool,omitempty" mapstructure:"cwl_tool,omitempty"`
	//CWL_tool_filename  string                    `bson:"cwl_tool_filename,omitempty" json:"cwl_tool_filename,omitempty" mapstructure:"cwl_tool_filename,omitempty"`
	Tool            interface{}               `bson:"tool,omitempty" json:"tool,omitempty" mapstructure:"tool,omitempty"`
	ToolFilename    string                    `bson:"tool_filename,omitempty" json:"tool_filename,omitempty" mapstructure:"tool_filename,omitempty"`
	Outputs         *cwl.Job_document         `bson:"outputs,omitempty" json:"outputs,omitempty" mapstructure:"outputs,omitempty"`
	OutputsExpected *[]cwl.WorkflowStepOutput `bson:"outputs_expected,omitempty" json:"outputs_expected,omitempty" mapstructure:"outputs_expected,omitempty"` // this is the subset of outputs that are needed by the workflow
	Notice          `bson:",inline" json:",inline" mapstructure:",squash"`
}

// NewCWLWorkunit _
func NewCWLWorkunit() *CWLWorkunit {
	return &CWLWorkunit{
		JobInput: nil,
		//CWL_tool:        nil,
		Tool:            nil,
		Outputs:         nil, // formerly Tool_results
		OutputsExpected: nil,
	}

}

// NewCWLWorkunitFromInterface _
func NewCWLWorkunitFromInterface(native interface{}, context *cwl.WorkflowContext) (workunit *CWLWorkunit, schemata []cwl.CWLType_Type, err error) {

	workunit = &CWLWorkunit{}

	switch native.(type) {

	case map[string]interface{}:

		native_map, ok := native.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewCWLWorkunitFromInterface) type error")
			return
		}

		jobInputGeneric, hasJobInputGeneric := native_map["job_input"]
		if hasJobInputGeneric {

			job_input, xerr := cwl.NewJob_documentFromNamedTypes(jobInputGeneric, context)
			if xerr != nil {
				err = fmt.Errorf("(NewCWLWorkunitFromInterface) NewJob_document failed: %s", xerr.Error())
				return
			}
			workunit.JobInput = job_input

		}

		workunit.JobInputFilename, _ = native_map["JobInput_filename"].(string)
		//workunit.CWL_tool_filename, _ = native_map["CWL_tool_filename"].(string)
		workunit.ToolFilename, _ = native_map["tool_filename"].(string)

		outputs_expected_generic, has_outputs_expected := native_map["outputs_expected"]
		if has_outputs_expected {
			if outputs_expected_generic != nil {
				outputs_expected, xerr := cwl.NewWorkflowStepOutputArray(outputs_expected_generic, context)
				if xerr != nil {
					err = fmt.Errorf("(NewCWLWorkunitFromInterface) NewWorkflowStepOutput failed: %s", xerr.Error())
					return
				}

				workunit.OutputsExpected = &outputs_expected
			}
		}

		tool_generic, has_tool_generic := native_map["tool"]
		if has_tool_generic {

			var schemata_new []cwl.CWLType_Type

			var class string
			class, err = cwl.GetClass(tool_generic)

			switch class {

			case "CommandLineTool":
				var commandlinetool *cwl.CommandLineTool

				commandlinetool, schemata_new, err = cwl.NewCommandLineTool(tool_generic, nil, context)
				if err != nil {
					err = fmt.Errorf("(NewCWLWorkunitFromInterface) NewCommandLineTool failed: %s", err.Error())
					return
				}
				workunit.Tool = commandlinetool

			case "ExpressionTool":
				var expressiontool *cwl.ExpressionTool

				expressiontool, err = cwl.NewExpressionTool(tool_generic, nil, nil, context)
				if err != nil {
					err = fmt.Errorf("(NewCWLWorkunitFromInterface) NewExpreassonTool failed: %s", err.Error())
					return
				}
				workunit.Tool = expressiontool
			default:
				err = fmt.Errorf("(NewCWLWorkunitFromInterface) class %s unknown", class)
				return
			}

			for i, _ := range schemata_new {
				schemata = append(schemata, schemata_new[i])
			}

		}

	default:
		err = fmt.Errorf("(NewCWLWorkunitFromInterface) wrong type, map expected")
		return

	}

	return

}
