package cwl

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
)

type CWLObject interface {
	IsCWLObject()
}

type CWLObjectArray []CWLObject

type CWLObjectImpl struct{}

func (c *CWLObjectImpl) IsCWLObject() {}

func NewCWLObject(original interface{}, injectedRequirements []Requirement, context *WorkflowContext) (obj CWLObject, schemata []CWLType_Type, err error) {
	//fmt.Println("(NewCWLObject) starting")

	if original == nil {
		err = fmt.Errorf("(NewCWLObject) original is nil!")
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

		class_thing, ok := elem["class"]
		if !ok {
			fmt.Println("------------------")
			spew.Dump(original)
			err = errors.New("(NewCWLObject) object has no field \"class\"")
			return
		}

		cwl_object_type, xok := class_thing.(string)

		if !xok {
			fmt.Println("------------------")
			spew.Dump(original)
			err = errors.New("(NewCWLObject) class is not astring")
			return
		}

		cwl_object_id := elem["id"].(string)
		if !ok {
			err = errors.New("(NewCWLObject) object has no member id")
			return
		}
		//fmt.Println("NewCWLObject C")
		switch cwl_object_type {
		case "CommandLineTool":
			//fmt.Println("NewCWLObject CommandLineTool")
			logger.Debug(1, "(NewCWLObject) parse CommandLineTool")

			fmt.Println("CommandLineTool interface:")
			spew.Dump(elem)

			var clt *CommandLineTool
			clt, schemata, err = NewCommandLineTool(elem, injectedRequirements, context)
			if err != nil {
				err = fmt.Errorf("(NewCWLObject) NewCommandLineTool returned: %s", err.Error())
				return
			}
			fmt.Println("CommandLineTool object:")
			spew.Dump(clt)
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
			obj, schemata, err = NewWorkflow(elem, injectedRequirements, context)
			if err != nil {

				err = fmt.Errorf("(NewCWLObject) NewWorkflow returned: %s", err.Error())
				return
			}

			return
		} // end switch

		cwl_type, xerr := NewCWLType(cwl_object_id, elem, context)
		if xerr != nil {

			err = fmt.Errorf("(NewCWLObject) NewCWLType returns %s", xerr.Error())
			return
		}
		obj = cwl_type
	default:
		//fmt.Printf("(NewCWLObject), unknown type %s\n", reflect.TypeOf(original))
		err = fmt.Errorf("(NewCWLObject), unknown type %s", reflect.TypeOf(original))
		return
	}
	//fmt.Println("(NewCWLObject) leaving")
	return
}

func NewCWLObjectArray(original interface{}, context *WorkflowContext) (array CWLObjectArray, schemata []CWLType_Type, err error) {

	//original, err = makeStringMap(original)
	//if err != nil {
	//	return
	//}

	array = CWLObjectArray{}

	switch original.(type) {

	case []interface{}:

		org_a := original.([]interface{})

		for _, element := range org_a {
			var schemataNew []CWLType_Type
			var cwl_object CWLObject
			cwl_object, schemataNew, err = NewCWLObject(element, nil, context)
			if err != nil {
				err = fmt.Errorf("(NewCWLObjectArray) NewCWLObject returned %s", err.Error())
				return
			}

			array = append(array, cwl_object)

			for i, _ := range schemataNew {
				schemata = append(schemata, schemataNew[i])
			}
		}

		return

	default:
		err = fmt.Errorf("(NewCWLObjectArray), unknown type %s", reflect.TypeOf(original))
	}
	return

}
