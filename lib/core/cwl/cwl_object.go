package cwl

import (
	"errors"
	"fmt"
	"path"
	"reflect"
	"strings"

	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
)

// CWLObject _
type CWLObject interface {
	IsCWLObject()
}

// CWLObjectArray _
type CWLObjectArray []CWLObject

// CWLObjectImpl _
type CWLObjectImpl struct{}

// IsCWLObject _
func (c *CWLObjectImpl) IsCWLObject() {}

// NewCWLObject _
//  objectIdentifier - is optional, should be used for object that are read from files, or embedded workflow/tool
// baseIdentifier is parent identifier
func NewCWLObject(original interface{}, objectIdentifier string, parentIdentifier string, injectedRequirements []Requirement, context *WorkflowContext) (obj CWLObject, schemata []CWLType_Type, err error) {
	//fmt.Println("(NewCWLObject) starting")

	if original == nil {
		err = fmt.Errorf("(NewCWLObject) original is nil! ")
		return
	}

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}
	//fmt.Println("NewCWLObject 2")
	switch original.(type) {
	case map[string]interface{}:
		elem := original.(map[string]interface{})
		//fmt.Println("NewCWLObject B")

		classThing, ok := elem["class"]
		if !ok {
			fmt.Println("------------------")
			spew.Dump(original)
			err = errors.New("(NewCWLObject) object has no field \"class\"")
			return
		}

		cwlObjectType, xok := classThing.(string)

		if !xok {
			fmt.Println("------------------")
			spew.Dump(original)
			err = errors.New("(NewCWLObject) class is not astring")
			return
		}

		var r Requirement

		r, err = NewRequirement(cwlObjectType, original, nil, context)
		if err == nil {
			obj = r
			return
		}

		identifierIf, hasIdentifier := elem["id"]
		if !hasIdentifier {

			if objectIdentifier == "" {
				err = errors.New("(NewCWLObject) object has no member id and objectIdentifier is empty")
				return
			}
			elem["id"] = objectIdentifier
			identifierIf = objectIdentifier
		}

		var cwlObjectIdentifer string
		cwlObjectIdentifer, ok = identifierIf.(string)
		if !ok {
			err = errors.New("(NewCWLObject) id is not a string")
			return
		}

		if !strings.HasPrefix(cwlObjectIdentifer, "#") {
			cwlObjectIdentifer = path.Join(parentIdentifier, cwlObjectIdentifer)
		}

		//fmt.Println("NewCWLObject C")
		switch cwlObjectType {
		case "CommandLineTool":
			//fmt.Println("NewCWLObject CommandLineTool")
			logger.Debug(1, "(NewCWLObject) parse CommandLineTool")

			//fmt.Println("CommandLineTool interface:")
			//spew.Dump(elem)

			var clt *CommandLineTool
			clt, schemata, err = NewCommandLineTool(elem, parentIdentifier, objectIdentifier, injectedRequirements, context)
			if err != nil {
				err = fmt.Errorf("(NewCWLObject) NewCommandLineTool returned: %s", err.Error())
				return
			}
			//fmt.Println("CommandLineTool object:")
			//spew.Dump(clt)
			obj = clt

			return
		case "ExpressionTool":
			//fmt.Println("NewCWLObject CommandLineTool")
			logger.Debug(1, "(NewCWLObject) parse ExpressionTool")
			var et *ExpressionTool
			et, err = NewExpressionTool(elem, nil, injectedRequirements, context) // TODO provide schemata
			if err != nil {
				err = fmt.Errorf("(NewCWLObject) ExpressionTool returned: %s", err.Error())
				return
			}

			obj = et

			return
		case "Workflow":
			//fmt.Println("NewCWLObject Workflow")
			logger.Debug(1, "parse Workflow")
			obj, schemata, err = NewWorkflow(elem, parentIdentifier, objectIdentifier, injectedRequirements, context)
			if err != nil {

				err = fmt.Errorf("(NewCWLObject) NewWorkflow returned: %s", err.Error())
				return
			}

			return
		} // end switch

		cwlType, xerr := NewCWLType(cwlObjectIdentifer, parentIdentifier, elem, context)
		if xerr != nil {

			err = fmt.Errorf("(NewCWLObject) NewCWLType returns %s", xerr.Error())
			return
		}
		obj = cwlType
	default:
		//fmt.Printf("(NewCWLObject), unknown type %s\n", reflect.TypeOf(original))
		err = fmt.Errorf("(NewCWLObject), unknown type %s", reflect.TypeOf(original))
		return
	}

	//fmt.Println("(NewCWLObject) leaving")
	return
}
