package cwl

import (
	"bytes"
	"fmt"

	"github.com/MG-RAST/AWE/lib/logger"
	//"github.com/davecgh/go-spew/spew"
	"reflect"
	"regexp"
	"strings"
)

type CWL_collection struct {
	Workflows          map[string]*Workflow
	WorkflowStepInputs map[string]*WorkflowStepInput
	CommandLineTools   map[string]*CommandLineTool
	ExpressionTools    map[string]*ExpressionTool

	Files    map[string]*File
	Strings  map[string]*String
	Ints     map[string]*Int
	Booleans map[string]*Boolean
	All      map[string]CWL_object // everything goes in here
	//Job_input          *Job_document
	Job_input_map *JobDocMap

	Schemata map[string]CWLType_Type
}

func (c CWL_collection) Evaluate(raw string) (parsed string) {

	reg := regexp.MustCompile(`\$\([\w.]+\)`) // https://github.com/google/re2/wiki/Syntax

	parsed = raw
	for {

		matches := reg.FindAll([]byte(parsed), -1)
		fmt.Printf("Matches: %d\n", len(matches))
		if len(matches) == 0 {
			return parsed
		}
		for _, match := range matches {
			key := bytes.TrimPrefix(match, []byte("$("))
			key = bytes.TrimSuffix(key, []byte(")"))

			// trimming of inputs. is only a work-around
			key = bytes.TrimPrefix(key, []byte("inputs."))

			value_str := ""
			value, err := c.GetString(string(key))

			if err != nil {
				value_str = "<ERROR_NOT_FOUND:" + string(key) + ">"
			} else {
				value_str = value.String()
			}

			logger.Debug(1, "evaluate %s -> %s\n", key, value_str)
			parsed = strings.Replace(parsed, string(match), value_str, 1)
		}

	}

}

func (c CWL_collection) AddSchemata(obj []CWLType_Type) (err error) {
	//fmt.Printf("(AddSchemata)\n")
	for i, _ := range obj {
		id := obj[i].GetId()
		if id == "" {
			err = fmt.Errorf("id empty")
			return
		}

		//fmt.Printf("Adding %s\n", id)

		_, ok := c.Schemata[id]
		if ok {
			return
		}

		c.Schemata[id] = obj[i]
	}
	return
}

func (c CWL_collection) GetSchemata() (obj []CWLType_Type, err error) {
	obj = []CWLType_Type{}
	for _, schema := range c.Schemata {
		obj = append(obj, schema)
	}
	return
}

func (c CWL_collection) AddArray(object_array []Named_CWL_object) (err error) {

	for i, _ := range object_array {
		pair := object_array[i]

		err = c.Add(pair.Id, pair.Value)
		if err != nil {
			return
		}

	}

	return

}

func (c CWL_collection) Add(id string, obj CWL_object) (err error) {

	if id == "" {
		err = fmt.Errorf("(CWL_collection/Add) id is empty")
		return
	}

	logger.Debug(3, "Adding object %s to collection", id)

	_, ok := c.All[id]
	if ok {
		err = fmt.Errorf("Object %s already in collection", id)
		return
	}

	switch obj.(type) {
	case *Workflow:
		c.Workflows[id] = obj.(*Workflow)
	case *WorkflowStepInput:
		c.WorkflowStepInputs[id] = obj.(*WorkflowStepInput)
	case *CommandLineTool:
		c.CommandLineTools[id] = obj.(*CommandLineTool)
	case *ExpressionTool:
		c.ExpressionTools[id] = obj.(*ExpressionTool)
	case *File:
		c.Files[id] = obj.(*File)
	case *String:
		c.Strings[id] = obj.(*String)
	case *Boolean:
		c.Booleans[id] = obj.(*Boolean)
	case *Int:
		obj_int, ok := obj.(*Int)
		if !ok {
			err = fmt.Errorf("could not make Int type assertion")
			return
		}
		c.Ints[id] = obj_int
	default:
		logger.Debug(3, "adding type %s to CWL_collection.All", reflect.TypeOf(obj))
	}
	c.All[id] = obj

	return
}

func (c CWL_collection) Get(id string) (obj CWL_object, err error) {
	obj, ok := c.All[id]
	if !ok {
		for k, _ := range c.All {
			logger.Debug(3, "collection: %s", k)
		}
		err = fmt.Errorf("(All) item %s not found in collection", id)
	}
	return
}

func (c CWL_collection) GetCWLType(id string) (obj CWLType, err error) {
	var ok bool
	obj, ok = c.Files[id]
	if ok {
		return
	}
	obj, ok = c.Strings[id]
	if ok {
		return
	}

	obj, ok = c.Ints[id]
	if ok {
		return
	}
	obj, ok = c.Booleans[id]
	if ok {
		return
	}

	err = fmt.Errorf("(GetType) %s not found", id)
	return

}

func (c CWL_collection) GetFile(id string) (obj *File, err error) {
	obj, ok := c.Files[id]
	if !ok {
		err = fmt.Errorf("(GetFile) item %s not found in collection", id)
	}
	return
}

func (c CWL_collection) GetString(id string) (obj *String, err error) {
	obj, ok := c.Strings[id]
	if !ok {
		err = fmt.Errorf("(GetString) item %s not found in collection", id)
	}
	return
}

func (c CWL_collection) GetInt(id string) (obj *Int, err error) {
	obj, ok := c.Ints[id]
	if !ok {
		err = fmt.Errorf("(GetInt) item %s not found in collection", id)
	}
	return
}

func (c CWL_collection) GetWorkflowStepInput(id string) (obj *WorkflowStepInput, err error) {
	obj, ok := c.WorkflowStepInputs[id]
	if !ok {
		err = fmt.Errorf("(GetWorkflowStepInput) item %s not found in collection", id)
	}
	return
}

func (c CWL_collection) GetCommandLineTool(id string) (obj *CommandLineTool, err error) {
	obj, ok := c.CommandLineTools[id]
	if !ok {
		err = fmt.Errorf("(GetCommandLineTool) item %s not found in collection", id)
	}
	return
}

func (c CWL_collection) GetExpressionTool(id string) (obj *ExpressionTool, err error) {
	obj, ok := c.ExpressionTools[id]
	if !ok {
		err = fmt.Errorf("(GetExpressionTool) item %s not found in collection", id)
	}
	return
}

func (c CWL_collection) GetWorkflow(id string) (obj *Workflow, err error) {
	obj, ok := c.Workflows[id]
	if !ok {
		err = fmt.Errorf("(GetWorkflow) item %s not found in collection", id)
	}
	return
}

func NewCWL_collection() (collection CWL_collection) {
	collection = CWL_collection{}

	collection.Workflows = make(map[string]*Workflow)
	collection.WorkflowStepInputs = make(map[string]*WorkflowStepInput)
	collection.CommandLineTools = make(map[string]*CommandLineTool)
	collection.ExpressionTools = make(map[string]*ExpressionTool)
	collection.Files = make(map[string]*File)
	collection.Strings = make(map[string]*String)
	collection.Ints = make(map[string]*Int)
	collection.Booleans = make(map[string]*Boolean)
	collection.All = make(map[string]CWL_object)
	collection.Schemata = make(map[string]CWLType_Type)
	return
}
