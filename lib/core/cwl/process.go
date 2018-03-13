package cwl

import (
	"fmt"
	"reflect"

	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
)

// needed for run in http://www.commonwl.org/v1.0/Workflow.html#WorkflowStep
// string | CommandLineTool | ExpressionTool | Workflow

type Process interface {
	CWL_object
	Is_process()
}

type ProcessPointer struct {
	Id    string
	Value string
}

func (p *ProcessPointer) Is_process() {}
func (p *ProcessPointer) GetClass() string {
	return "ProcessPointer"
}
func (p *ProcessPointer) GetId() string   { return p.Id }
func (p *ProcessPointer) SetId(string)    {}
func (p *ProcessPointer) Is_CWL_minimal() {}

func NewProcessPointer(original interface{}) (pp *ProcessPointer, err error) {

	switch original.(type) {
	case map[string]interface{}:
		//original_map, ok := original.(map[string]interface{})

		pp = &ProcessPointer{}

		err = mapstructure.Decode(original, pp)
		if err != nil {
			err = fmt.Errorf("(NewCommandInputParameter) decode error: %s", err.Error())
			return
		}
		return
	default:
		spew.Dump(original)
		err = fmt.Errorf("(NewProcess) type %s unknown", reflect.TypeOf(original))
	}
	return
}

// returns CommandLineTool, ExpressionTool or Workflow
func NewProcess(original interface{}) (process interface{}, schemata []CWLType_Type, err error) {

	logger.Debug(3, "NewProcess starting")

	original, err = MakeStringMap(original)
	if err != nil {
		return
	}

	switch original.(type) {
	case string:
		original_str := original.(string)

		//pp := &ProcessPointer{Value: original_str}

		process = original_str
		return
	case map[string]interface{}:
		original_map, ok := original.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewProcess) failed")
			return
		}

		var class string
		class, err = GetClass(original_map)
		if err != nil {
			err = fmt.Errorf("(NewProcess) GetClass returned: %s", err.Error())
			return
		}

		switch class {
		//case "":
		//return NewProcessPointer(original)
		case "Workflow":
			return NewWorkflow(original)
		case "Expression":
			process, err = NewExpression(original)
			return
		case "CommandLineTool":
			process, schemata, err = NewCommandLineTool(original)
			return
		case "ExpressionTool":
			err = fmt.Errorf("(NewProcess) ExpressionTool not implemented yet")
			return
		default:
			err = fmt.Errorf("(NewProcess) class %s not supported", class)
			return
		}

	default:
		spew.Dump(original)
		err = fmt.Errorf("(NewProcess) type %s unknown", reflect.TypeOf(original))

	}

	return
}
