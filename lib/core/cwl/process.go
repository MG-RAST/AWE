package cwl

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"

	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
)

// needed for run in http://www.commonwl.org/v1.0/Workflow.html#WorkflowStep
// string | CommandLineTool | ExpressionTool | Workflow

// Process _
type Process interface {
	CWLObject
	IdentifierInterface
	IsProcess()
}

// // ProcessPointer _
// type ProcessPointer struct {
// 	ID    string
// 	Value string
// }

// // IsProcess _
// func (p *ProcessPointer) IsProcess() {}

// // GetClass _
// func (p *ProcessPointer) GetClass() string {
// 	return "ProcessPointer"
// }

// // GetID _
// func (p *ProcessPointer) GetID() string { return p.ID }

// // SetID _
// func (p *ProcessPointer) SetID(string) {}

// // IsCWLMinimal _
// func (p *ProcessPointer) IsCWLMinimal() {}

// NewProcessPointer _
// func NewProcessPointer(original interface{}) (pp *ProcessPointer, err error) {

// 	switch original.(type) {
// 	case map[string]interface{}:
// 		//original_map, ok := original.(map[string]interface{})

// 		pp = &ProcessPointer{}

// 		err = mapstructure.Decode(original, pp)
// 		if err != nil {
// 			err = fmt.Errorf("(NewCommandInputParameter) decode error: %s", err.Error())
// 			return
// 		}
// 		return
// 	default:
// 		spew.Dump(original)
// 		err = fmt.Errorf("(NewProcess) type %s unknown", reflect.TypeOf(original))
// 	}
// 	return
// }

// NewProcess returns CommandLineTool, ExpressionTool or Workflow
func NewProcess(original interface{}, parentID string, processID string, injectedRequirements []Requirement, context *WorkflowContext) (process Process, schemata []CWLType_Type, err error) {

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
		err = fmt.Errorf("(NewProcess) MakeStringMap returned: %s", err.Error())
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
			var processIf interface{}
			processIf, ok = context.Objects[processIdentifer]
			if ok {
				if processIf == nil {
					err = fmt.Errorf("(NewProcess) processIf == nil")
					return
				}
				process, ok = processIf.(Process)
				if !ok {
					err = fmt.Errorf("(NewProcess) not a Process")
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

			object, schemata, err = NewCWLObject(processIf, processID, parentID, injectedRequirements, context)
			if err != nil {
				err = fmt.Errorf("(NewProcess) A NewCWLObject returns %s", err.Error())
				return
			}

			context.Objects[processIdentifer] = object
			process, ok = object.(Process)
			if !ok {
				err = fmt.Errorf("(NewProcess) not a Process")
				return
			}

			return
		}

		ifObjectsStr := ""
		for id := range context.IfObjects {
			//fmt.Printf("Id: %s\n", id)
			ifObjectsStr += "," + id
		}
		// for id := range context.Objects {
		// 	fmt.Printf("Id: %s\n", id)
		// }

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

		processIf, ok, err = context.Get(processIdentifer, true)
		if err != nil {
			err = fmt.Errorf("(NewProcess) context.Get returned: %s", err.Error())
			return
		}
		if ok {
			process, ok = processIf.(Process)
			if !ok {
				err = fmt.Errorf("(NewProcess) not a Process")
				return
			}

			return
		}

		err = fmt.Errorf("(NewProcess) Process %s not found (tried newFileToLoad: %s) parentID: %s", processIdentifer, newFileToLoad, parentID)

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

		if parentID == "" {
			err = fmt.Errorf("(NewProcess) parentID empty !?")
			return
		}

		switch class {
		//case "":
		//return NewProcessPointer(original)
		case "Workflow":
			process, schemata, err = NewWorkflow(original, parentID, processID, injectedRequirements, context)
			if err != nil {
				err = fmt.Errorf("(NewProcess) NewWorkflow returned: %s", err.Error())
				return
			}
			return
		// case "Expression":
		// 	process, err = NewExpression(original)
		// 	if err != nil {
		// 		err = fmt.Errorf("(NewProcess) NewExpression returned: %s", err.Error())
		// 		return
		// 	}
		// 	return
		case "CommandLineTool":
			process, schemata, err = NewCommandLineTool(original, parentID, processID, injectedRequirements, context) // TODO merge schemata correctly !
			if err != nil {
				err = fmt.Errorf("(NewProcess) NewCommandLineTool returned: %s (processID=%s)", err.Error(), processID)
				return
			}
			return
		case "ExpressionTool":
			process, err = NewExpressionTool(original, parentID, processID, schemata, injectedRequirements, context)
			if err != nil {
				err = fmt.Errorf("(NewProcess) NewExpressionTool returned: %s", err.Error())
				return
			}
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
