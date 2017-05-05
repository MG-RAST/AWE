package cwl

import (
	"bytes"
	"fmt"
	"github.com/MG-RAST/AWE/lib/logger"
	"regexp"
	"strings"
)

type CWL_collection struct {
	Workflows          map[string]*Workflow
	WorkflowStepInputs map[string]*WorkflowStepInput
	CommandLineTools   map[string]*CommandLineTool
	Files              map[string]*File
	Strings            map[string]*String
	Ints               map[string]*Int
	Booleans           map[string]*Boolean
	All                map[string]*CWL_object
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

func (c CWL_collection) Add(obj CWL_object) (err error) {

	id := obj.GetId()

	if id == "" {
		err = fmt.Errorf("key is empty")
		return
	}

	switch obj.GetClass() {
	case "Workflow":
		c.Workflows[id] = obj.(*Workflow)
	case "WorkflowStepInput":
		c.WorkflowStepInputs[id] = obj.(*WorkflowStepInput)
	case "CommandLineTool":
		c.CommandLineTools[id] = obj.(*CommandLineTool)
	case "File":
		c.Files[id] = obj.(*File)
	case "String":
		c.Strings[id] = obj.(*String)
	case "Boolean":
		c.Booleans[id] = obj.(*Boolean)
	case "Int":
		obj_int, ok := obj.(*Int)
		if !ok {
			err = fmt.Errorf("could not make Int type assertion")
			return
		}
		c.Ints[id] = obj_int
	}
	c.All[id] = &obj

	return
}

func (c CWL_collection) Get(id string) (obj *CWL_object, err error) {
	obj, ok := c.All[id]
	if !ok {
		err = fmt.Errorf("item %s not found in collection", id)
	}
	return
}

func (c CWL_collection) GetFile(id string) (obj *File, err error) {
	obj, ok := c.Files[id]
	if !ok {
		err = fmt.Errorf("item %s not found in collection", id)
	}
	return
}

func (c CWL_collection) GetString(id string) (obj *String, err error) {
	obj, ok := c.Strings[id]
	if !ok {
		err = fmt.Errorf("item %s not found in collection", id)
	}
	return
}

func (c CWL_collection) GetInt(id string) (obj *Int, err error) {
	obj, ok := c.Ints[id]
	if !ok {
		err = fmt.Errorf("item %s not found in collection", id)
	}
	return
}

func (c CWL_collection) GetWorkflowStepInput(id string) (obj *WorkflowStepInput, err error) {
	obj, ok := c.WorkflowStepInputs[id]
	if !ok {
		err = fmt.Errorf("item %s not found in collection", id)
	}
	return
}

func NewCWL_collection() (collection CWL_collection) {
	collection = CWL_collection{}

	collection.Workflows = make(map[string]*Workflow)
	collection.WorkflowStepInputs = make(map[string]*WorkflowStepInput)
	collection.CommandLineTools = make(map[string]*CommandLineTool)
	collection.Files = make(map[string]*File)
	collection.Strings = make(map[string]*String)
	collection.Ints = make(map[string]*Int)
	collection.Booleans = make(map[string]*Boolean)
	collection.All = make(map[string]*CWL_object)
	return
}
