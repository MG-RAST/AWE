package cwl

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"

	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
)

// needed for run in http://www.commonwl.org/v1.0/Workflow.html#WorkflowStep
// string | CommandLineTool | ExpressionTool | Workflow

type Process interface {
	CWLObject
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
func (p *ProcessPointer) GetID() string { return p.Id }
func (p *ProcessPointer) SetID(string)  {}
func (p *ProcessPointer) IsCWLMinimal() {}

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

// NewProcess returns CommandLineTool, ExpressionTool or Workflow
func NewProcess(original interface{}, processID string, injectedRequirements []Requirement, context *WorkflowContext) (process interface{}, schemata []CWLType_Type, err error) {

	//logger.Debug(3, "(NewProcess) starting")
	if context == nil {
		err = fmt.Errorf("(NewProcess) context == nil")
		return
	}

	if context.CwlVersion == "" {
		err = fmt.Errorf("(NewProcess) CwlVersion empty")
		return
	}

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	switch original.(type) {
	case string:
		//logger.Debug(3, "(NewProcess) a string")
		originalStr := original.(string)

		processIdentifer := originalStr
		if !strings.HasPrefix(originalStr, "#") {
			processIdentifer = "#" + originalStr
		}

		//pp := &ProcessPointer{Value: originalStr}
		//process = originalStr
		var ok bool

		if context.Objects != nil {

			process, ok = context.Objects[processIdentifer]
			if ok {
				if process == nil {
					err = fmt.Errorf("(NewProcess) process == nil")
					return
				}

				//logger.Debug(3, "(NewProcess) object %s found", originalStr)
				// refrenced object has already been parsed once
				// TODO may want to parse a second time and make a new copy
				return
			}
		}
		var processIf interface{}
		processIf, ok = context.IfObjects[processIdentifer]
		if ok {
			logger.Debug(3, "(NewProcess) %s found in object_if", processIdentifer)
			var object CWLObject

			object, schemata, err = NewCWLObject(processIf, "", processID, injectedRequirements, context)
			if err != nil {
				err = fmt.Errorf("(NewProcess) A NewCWLObject returns %s", err.Error())
				return
			}

			context.Objects[processIdentifer] = object
			process = object

			return
		}

		ifObjectsStr := ""
		for id := range context.IfObjects {
			fmt.Printf("Id: %s\n", id)
			ifObjectsStr += "," + id
		}
		for id := range context.Objects {
			fmt.Printf("Id: %s\n", id)
		}

		//originalStrArray := strings.Split(originalStr, "#")
		fileFromStr := strings.TrimPrefix(processIdentifer, "#")
		if context.Path == "" || context.Path == "-" {
			err = fmt.Errorf("(NewProcess) context.Path is not set, there should be no files to load (processIdentifer %s not found)", processIdentifer)
			return
		}
		newFileToLoad := path.Join(context.Path, fileFromStr) // TODO context.Path is not correct, depends on location of parent document
		newFileToLoadBase := path.Base(newFileToLoad)
		entrypoint := ""
		// if len(originalStrArray) > 1 {
		// 	entrypoint = originalStrArray[1]
		// }

		_, err = os.Stat(newFileToLoad)
		if err != nil {

			err = fmt.Errorf("(NewProcess) \"%s\" not found in context (found %s) and does not seem to be a file (processIdentifer: %s, fileFromStr: %s)", newFileToLoad, ifObjectsStr, processIdentifer, fileFromStr)
			return
		}

		//var newObjectArray []NamedCWLObject
		var schemas []interface{}
		processIf, schemata, _, schemas, _, err = ParseCWLDocumentFile(context, newFileToLoad, entrypoint, context.Path, newFileToLoadBase)
		if err != nil {

			err = fmt.Errorf("(NewProcess) ParseCWLDocumentFile returned: %s", err.Error())
			return
		}

		_ = schemas
		// for i := range newObjectArray {
		// 	thing := newObjectArray[i]

		// 	err = context.Add(thing.Id, thing.Value, "NewProcess")
		// 	if err != nil {

		// 		err = fmt.Errorf("(NewProcess) context.Add returned: %s", err.Error())
		// 		return
		// 	}
		// }

		process, ok, err = context.Get(processIdentifer, true)
		if err != nil {
			err = fmt.Errorf("(NewProcess) context.Get returned: %s", err.Error())
			return
		}
		if ok {
			if process == nil {
				err = fmt.Errorf("(NewProcess) B process == nil")
				return
			}
			return
		}

		err = fmt.Errorf("(NewProcess) Process %s not found ", processIdentifer)

		return
	case map[string]interface{}:
		originalMap, ok := original.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewProcess) failed")
			return
		}

		var class string
		class, err = GetClass(originalMap)
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
			process, schemata, err = NewCommandLineTool(original, processID, processID, injectedRequirements, context) // TODO merge schemata correctly !
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
