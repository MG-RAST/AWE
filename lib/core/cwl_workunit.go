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
func NewCWLWorkunitFromInterface(native interface{}, baseIdentifier string, context *cwl.WorkflowContext) (workunit *CWLWorkunit, schemata []cwl.CWLType_Type, err error) {

	workunit = &CWLWorkunit{}

	switch native.(type) {

	case map[string]interface{}:

		nativeMap, ok := native.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewCWLWorkunitFromInterface) type error")
			return
		}

		jobInputGeneric, hasJobInputGeneric := nativeMap["job_input"]
		if hasJobInputGeneric {

			jobInput, xerr := cwl.NewJob_documentFromNamedTypes(jobInputGeneric, context)
			if xerr != nil {
				err = fmt.Errorf("(NewCWLWorkunitFromInterface) NewJob_document failed: %s", xerr.Error())
				return
			}
			workunit.JobInput = jobInput

		}

		workunit.JobInputFilename, _ = nativeMap["JobInput_filename"].(string)
		//workunit.CWL_tool_filename, _ = nativeMap["CWL_tool_filename"].(string)
		workunit.ToolFilename, _ = nativeMap["tool_filename"].(string)

		outputsExpectedGeneric, hasOutputsExpected := nativeMap["outputs_expected"]
		if hasOutputsExpected {
			if outputsExpectedGeneric != nil {
				outputsExpected, xerr := cwl.NewWorkflowStepOutputArray(outputsExpectedGeneric, context)
				if xerr != nil {
					err = fmt.Errorf("(NewCWLWorkunitFromInterface) NewWorkflowStepOutput failed: %s", xerr.Error())
					return
				}

				workunit.OutputsExpected = &outputsExpected
			}
		}

		toolGeneric, hasToolGeneric := nativeMap["tool"]
		if hasToolGeneric {

			var schemataNew []cwl.CWLType_Type

			var class string
			class, err = cwl.GetClass(toolGeneric)

			switch class {

			case "CommandLineTool":
				var commandlinetool *cwl.CommandLineTool

				commandlinetool, schemataNew, err = cwl.NewCommandLineTool(toolGeneric, baseIdentifier, "", nil, context)
				if err != nil {
					err = fmt.Errorf("(NewCWLWorkunitFromInterface) NewCommandLineTool failed: %s", err.Error())
					return
				}
				workunit.Tool = commandlinetool

			case "ExpressionTool":
				var expressiontool *cwl.ExpressionTool

				expressiontool, err = cwl.NewExpressionTool(toolGeneric, nil, nil, context)
				if err != nil {
					err = fmt.Errorf("(NewCWLWorkunitFromInterface) NewExpreassonTool failed: %s", err.Error())
					return
				}
				workunit.Tool = expressiontool
			default:
				err = fmt.Errorf("(NewCWLWorkunitFromInterface) class %s unknown", class)
				return
			}

			for i := range schemataNew {
				schemata = append(schemata, schemataNew[i])
			}

		}

	default:
		err = fmt.Errorf("(NewCWLWorkunitFromInterface) wrong type, map expected")
		return

	}

	return

}
