package cwl

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
)

type CWL_object interface {
	Is_CWL_object()
}

type CWL_object_array []CWL_object

type CWL_object_Impl struct{}

func (c *CWL_object_Impl) Is_CWL_object() {}

func New_CWL_object(original interface{}, injectedRequirements []Requirement, context *WorkflowContext) (obj CWL_object, schemata []CWLType_Type, err error) {
	//fmt.Println("(New_CWL_object) starting")

	if original == nil {
		err = fmt.Errorf("(New_CWL_object) original is nil!")
		return
	}

	original, err = MakeStringMap(original)
	if err != nil {
		return
	}
	//fmt.Println("New_CWL_object 2")
	switch original.(type) {
	case map[string]interface{}:
		elem := original.(map[string]interface{})
		//fmt.Println("New_CWL_object B")

		class_thing, ok := elem["class"]
		if !ok {
			fmt.Println("------------------")
			spew.Dump(original)
			err = errors.New("(New_CWL_object) object has no field \"class\"")
			return
		}

		cwl_object_type, xok := class_thing.(string)

		if !xok {
			fmt.Println("------------------")
			spew.Dump(original)
			err = errors.New("(New_CWL_object) class is not astring")
			return
		}

		cwl_object_id := elem["id"].(string)
		if !ok {
			err = errors.New("(New_CWL_object) object has no member id")
			return
		}
		//fmt.Println("New_CWL_object C")
		switch cwl_object_type {
		case "CommandLineTool":
			//fmt.Println("New_CWL_object CommandLineTool")
			logger.Debug(1, "(New_CWL_object) parse CommandLineTool")
			var clt *CommandLineTool
			clt, schemata, err = NewCommandLineTool(elem, injectedRequirements, context)
			if err != nil {
				err = fmt.Errorf("(New_CWL_object) NewCommandLineTool returned: %s", err.Error())
				return
			}

			obj = clt

			return
		case "ExpressionTool":
			//fmt.Println("New_CWL_object CommandLineTool")
			logger.Debug(1, "(New_CWL_object) parse ExpressionTool")
			var et *ExpressionTool
			et, err = NewExpressionTool(elem, nil, injectedRequirements, context) // TODO provide schemata
			if err != nil {
				err = fmt.Errorf("(New_CWL_object) ExpressionTool returned: %s", err.Error())
				return
			}

			obj = et

			return
		case "Workflow":
			//fmt.Println("New_CWL_object Workflow")
			logger.Debug(1, "parse Workflow")
			obj, schemata, err = NewWorkflow(elem, injectedRequirements, context)
			if err != nil {

				err = fmt.Errorf("(New_CWL_object) NewWorkflow returned: %s", err.Error())
				return
			}

			return
		} // end switch

		cwl_type, xerr := NewCWLType(cwl_object_id, elem)
		if xerr != nil {

			err = fmt.Errorf("(New_CWL_object) NewCWLType returns %s", xerr.Error())
			return
		}
		obj = cwl_type
	default:
		//fmt.Printf("(New_CWL_object), unknown type %s\n", reflect.TypeOf(original))
		err = fmt.Errorf("(New_CWL_object), unknown type %s", reflect.TypeOf(original))
		return
	}
	//fmt.Println("(New_CWL_object) leaving")
	return
}

func NewCWL_object_array(original interface{}, context *WorkflowContext) (array CWL_object_array, schemata []CWLType_Type, err error) {

	//original, err = makeStringMap(original)
	//if err != nil {
	//	return
	//}

	array = CWL_object_array{}

	switch original.(type) {

	case []interface{}:

		org_a := original.([]interface{})

		for _, element := range org_a {
			var schemata_new []CWLType_Type
			var cwl_object CWL_object
			cwl_object, schemata_new, err = New_CWL_object(element, nil, context)
			if err != nil {
				err = fmt.Errorf("(NewCWL_object_array) New_CWL_object returned %s", err.Error())
				return
			}

			array = append(array, cwl_object)

			for i, _ := range schemata_new {
				schemata = append(schemata, schemata_new[i])
			}
		}

		return

	default:
		err = fmt.Errorf("(NewCWL_object_array), unknown type %s", reflect.TypeOf(original))
	}
	return

}
