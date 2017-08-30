package cwl

import (
	"bytes"
	"fmt"
	cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	"github.com/MG-RAST/AWE/lib/logger"
	"regexp"
	"strings"
)

type CWL_collection struct {
	Workflows          map[string]*Workflow
	WorkflowStepInputs map[string]*WorkflowStepInput
	CommandLineTools   map[string]*CommandLineTool
	Files              map[string]*cwl_types.File
	Strings            map[string]*cwl_types.String
	Ints               map[string]*cwl_types.Int
	Booleans           map[string]*cwl_types.Boolean
	All                map[string]*cwl_types.CWL_object // everything goes in here
	//Job_input          *Job_document
	Job_input_map *map[string]cwl_types.CWLType
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

func (c CWL_collection) Add(obj cwl_types.CWL_object) (err error) {

	id := obj.GetId()

	if id == "" {
		err = fmt.Errorf("key is empty")
		return
	}

	logger.Debug(3, "Adding object %s to collection", id)

	_, ok := c.All[id]
	if ok {
		err = fmt.Errorf("Object %s already in collection", id)
		return
	}
	//id = strings.TrimPrefix(id, "#")

	switch obj.GetClass() {
	case "Workflow":
		c.Workflows[id] = obj.(*Workflow)
	case "WorkflowStepInput":
		c.WorkflowStepInputs[id] = obj.(*WorkflowStepInput)
	case "CommandLineTool":
		c.CommandLineTools[id] = obj.(*CommandLineTool)
	case cwl_types.CWL_File:
		c.Files[id] = obj.(*cwl_types.File)
	case cwl_types.CWL_string:
		c.Strings[id] = obj.(*cwl_types.String)
	case cwl_types.CWL_boolean:
		c.Booleans[id] = obj.(*cwl_types.Boolean)
	case cwl_types.CWL_int:
		obj_int, ok := obj.(*cwl_types.Int)
		if !ok {
			err = fmt.Errorf("could not make Int type assertion")
			return
		}
		c.Ints[id] = obj_int
	}
	c.All[id] = &obj

	return
}

func (c CWL_collection) Get(id string) (obj *cwl_types.CWL_object, err error) {
	obj, ok := c.All[id]
	if !ok {
		for k, _ := range c.All {
			logger.Debug(3, "collection: %s", k)
		}
		err = fmt.Errorf("(All) item %s not found in collection", id)
	}
	return
}

func (c CWL_collection) GetType(id string) (obj cwl_types.CWLType, err error) {
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

func (c CWL_collection) GetFile(id string) (obj *cwl_types.File, err error) {
	obj, ok := c.Files[id]
	if !ok {
		err = fmt.Errorf("(GetFile) item %s not found in collection", id)
	}
	return
}

func (c CWL_collection) GetString(id string) (obj *cwl_types.String, err error) {
	obj, ok := c.Strings[id]
	if !ok {
		err = fmt.Errorf("(GetString) item %s not found in collection", id)
	}
	return
}

func (c CWL_collection) GetInt(id string) (obj *cwl_types.Int, err error) {
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

func NewCWL_collection() (collection CWL_collection) {
	collection = CWL_collection{}

	collection.Workflows = make(map[string]*Workflow)
	collection.WorkflowStepInputs = make(map[string]*WorkflowStepInput)
	collection.CommandLineTools = make(map[string]*CommandLineTool)
	collection.Files = make(map[string]*cwl_types.File)
	collection.Strings = make(map[string]*cwl_types.String)
	collection.Ints = make(map[string]*cwl_types.Int)
	collection.Booleans = make(map[string]*cwl_types.Boolean)
	collection.All = make(map[string]*cwl_types.CWL_object)
	return
}
