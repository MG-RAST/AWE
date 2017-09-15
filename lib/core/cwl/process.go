package cwl

import (
	"fmt"
	cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
	"reflect"
)

// needed for run in http://www.commonwl.org/v1.0/Workflow.html#WorkflowStep
// string | CommandLineTool | ExpressionTool | Workflow

type Process interface {
	cwl_types.CWL_object
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
func NewProcess(original interface{}) (process interface{}, err error) {

	original, err = makeStringMap(original)
	if err != nil {
		return
	}

	switch original.(type) {
	case string:
		original_str := original.(string)

		pp := &ProcessPointer{Value: original_str}

		process = pp
		return
	case map[string]interface{}:
		original_map, ok := original.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewProcess) failed")
			return
		}

		var class cwl_types.CWLType_Type
		class, err = cwl_types.GetClass(original_map)
		if err != nil {
			err = fmt.Errorf("(NewProcess) cwl_types.GetClass returned: %s", err.Error())
			return
		}

		switch class {
		//case "":
		//return NewProcessPointer(original)
		case CWL_Workflow:
			return NewWorkflow(original)
		case cwl_types.CWL_Expression:
			return cwl_types.NewExpression(original)

		}

	default:
		spew.Dump(original)
		err = fmt.Errorf("(NewProcess) type %s unknown", reflect.TypeOf(original))

	}

	return
}
