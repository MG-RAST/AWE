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
func NewProcess(original interface{}, CwlVersion CWLVersion, injectedRequirements []Requirement, context *WorkflowContext) (process interface{}, schemata []CWLType_Type, err error) {

	//logger.Debug(3, "(NewProcess) starting")
	if context == nil {
		err = fmt.Errorf("(NewProcess) context == nil")
		return
	}

	if CwlVersion == "" {
		err = fmt.Errorf("(NewProcess) CwlVersion empty")
		return
	}

	original, err = MakeStringMap(original)
	if err != nil {
		return
	}

	switch original.(type) {
	case string:
		//logger.Debug(3, "(NewProcess) a string")
		original_str := original.(string)

		//pp := &ProcessPointer{Value: original_str}
		process = original_str
		var ok bool

		if context.Objects != nil {

			_, ok = context.Objects[original_str]
			if ok {
				//logger.Debug(3, "(NewProcess) object %s found", original_str)
				// refrenced object has already been parsed once
				// TODO may want to parse a second time and make a new copy
				return
			}
		}
		var process_if interface{}
		process_if, ok = context.If_objects[original_str]
		if ok {
			logger.Debug(3, "(NewProcess) %s found in object_if", original_str)
			var object CWL_object

			object, schemata, err = New_CWL_object(process_if, injectedRequirements, context)
			if err != nil {
				err = fmt.Errorf("(NewProcess) A New_CWL_object returns %s", err.Error())
				return
			}

			context.Objects[original_str] = object
			return
		}

		for id, _ := range context.If_objects {
			fmt.Printf("Id: %s\n", id)
		}
		for id, _ := range context.Objects {
			fmt.Printf("Id: %s\n", id)
		}
		err = fmt.Errorf("(NewProcess) %s not found in context", original_str)
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
			process, schemata, err = NewWorkflow(original, injectedRequirements, context)

			return
		case "Expression":
			process, err = NewExpression(original)
			return
		case "CommandLineTool":
			process, schemata, err = NewCommandLineTool(original, injectedRequirements, context) // TODO merge schemata correctly !
			return
		case "ExpressionTool":
			process, err = NewExpressionTool(original, schemata, injectedRequirements, context)
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
