package cwl

import (
	"fmt"
)

type CWL_collection struct {
	Workflows        map[string]Workflow
	CommandLineTools map[string]CommandLineTool
	Files            map[string]File
	Strings          map[string]string
	All              map[string]*CWL_object
}

func (c CWL_collection) Add(obj CWL_object) (err error) {

	id := obj.GetId()

	if id == "" {
		err = fmt.Errorf("key is empty")
	}

	switch obj.GetClass() {
	case "Workflow":
		c.Workflows[id] = obj.(Workflow)
	case "CommandLineTool":
		c.CommandLineTools[id] = obj.(CommandLineTool)
	case "File":
		c.Files[id] = obj.(File)
	}
	c.All[obj.GetId()] = &obj

	return
}

func (c CWL_collection) Get(id string) (obj *CWL_object, err error) {
	obj, ok := c.All[id]
	if !ok {
		err = fmt.Errorf("item %s not found in collection", id)
	}
	return
}

func (c CWL_collection) GetFile(id string) (obj File, err error) {
	obj, ok := c.Files[id]
	if !ok {
		err = fmt.Errorf("item %s not found in collection", id)
	}
	return
}

func NewCWL_collection() (collection CWL_collection) {
	collection = CWL_collection{}

	collection.Workflows = make(map[string]Workflow)
	collection.CommandLineTools = make(map[string]CommandLineTool)
	collection.Files = make(map[string]File)
	collection.All = make(map[string]*CWL_object)
	return
}
