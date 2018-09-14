package cwl

import (
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
)

type WorkflowContext struct {
	CWL_document `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // fields: CwlVersion, Base, Graph, Namespaces, Schemas
	Path         string
	//Namespaces   map[string]string
	//CWLVersion
	//CwlVersion CWLVersion    `yaml:"cwl_version"  json:"cwl_version" bson:"cwl_version" mapstructure:"cwl_version"`
	//CWL_graph  []interface{} `yaml:"cwl_graph"  json:"cwl_graph" bson:"cwl_graph" mapstructure:"cwl_graph"`
	// old ParsingContext
	If_objects map[string]interface{}
	Objects    map[string]CWL_object

	Workflows          map[string]*Workflow
	WorkflowStepInputs map[string]*WorkflowStepInput
	CommandLineTools   map[string]*CommandLineTool
	ExpressionTools    map[string]*ExpressionTool
	Files              map[string]*File
	Strings            map[string]*String
	Ints               map[string]*Int
	Booleans           map[string]*Boolean
	All                map[string]CWL_object // everything goes in here
	//Job_input          *Job_document
	Job_input_map *JobDocMap

	Schemata map[string]CWLType_Type
}

func (context *WorkflowContext) FillMaps(graph []interface{}) (err error) {

	if context.CwlVersion == "" {
		err = fmt.Errorf("(FillMaps) context.CwlVersion ==nil")
		return
	}

	//context := &WorkflowContext{}
	context.If_objects = make(map[string]interface{})
	context.Objects = make(map[string]CWL_object)

	if len(graph) == 0 {
		err = fmt.Errorf("(FillMaps) len(graph) == 0")
		return
	}

	// put interface objetcs into map: populate context.If_objects
	for _, elem := range graph {

		var id string
		id, err = GetId(elem)
		if err != nil {
			fmt.Println("object without id:")
			spew.Dump(elem)
			return
		}
		//fmt.Printf("id=\"%s\\n", id)

		context.If_objects[id] = elem

	}

	main_if, has_main := context.If_objects["#main"]
	if !has_main {
		var keys string
		for key, _ := range context.If_objects {
			keys += "," + key
		}
		err = fmt.Errorf("(FillMaps) #main not found in graph (found %s)", keys)
		return
	}

	// start with #main
	// recursivly add objects to context
	var object CWL_object
	var schemata_new []CWLType_Type
	object, schemata_new, err = New_CWL_object(main_if, nil, context)
	if err != nil {
		err = fmt.Errorf("(FillMaps) A New_CWL_object returned %s", err.Error())
		return
	}
	context.Objects["#main"] = object

	err = context.AddSchemata(schemata_new)
	if err != nil {
		err = fmt.Errorf("(FillMaps) context.AddSchemata returned %s", err.Error())
		return
	}
	//for i, _ := range schemata_new {
	//	schemata = append(schemata, schemata_new[i])
	//}

	return
}

func (c WorkflowContext) Evaluate(raw string) (parsed string) {

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

func (c WorkflowContext) AddSchemata(obj []CWLType_Type) (err error) {
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

func (c WorkflowContext) GetSchemata() (obj []CWLType_Type, err error) {
	obj = []CWLType_Type{}
	for _, schema := range c.Schemata {
		obj = append(obj, schema)
	}
	return
}

func (c WorkflowContext) AddArray(object_array []Named_CWL_object) (err error) {

	for i, _ := range object_array {
		pair := object_array[i]

		err = c.Add(pair.Id, pair.Value)
		if err != nil {
			return
		}

	}

	return

}

func (c WorkflowContext) Add(id string, obj CWL_object) (err error) {

	if id == "" {
		err = fmt.Errorf("(WorkflowContext/Add) id is empty")
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
		logger.Debug(3, "adding type %s to WorkflowContext.All", reflect.TypeOf(obj))
	}
	c.All[id] = obj

	return
}

func (c WorkflowContext) Get(id string) (obj CWL_object, err error) {
	obj, ok := c.All[id]
	if !ok {
		for k, _ := range c.All {
			logger.Debug(3, "collection: %s", k)
		}
		err = fmt.Errorf("(All) item %s not found in collection", id)
	}
	return
}

func (c WorkflowContext) GetCWLType(id string) (obj CWLType, err error) {
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

func (c WorkflowContext) GetFile(id string) (obj *File, err error) {
	obj, ok := c.Files[id]
	if !ok {
		err = fmt.Errorf("(GetFile) item %s not found in collection", id)
	}
	return
}

func (c WorkflowContext) GetString(id string) (obj *String, err error) {
	obj, ok := c.Strings[id]
	if !ok {
		err = fmt.Errorf("(GetString) item %s not found in collection", id)
	}
	return
}

func (c WorkflowContext) GetInt(id string) (obj *Int, err error) {
	obj, ok := c.Ints[id]
	if !ok {
		err = fmt.Errorf("(GetInt) item %s not found in collection", id)
	}
	return
}

func (c WorkflowContext) GetWorkflowStepInput(id string) (obj *WorkflowStepInput, err error) {
	obj, ok := c.WorkflowStepInputs[id]
	if !ok {
		err = fmt.Errorf("(GetWorkflowStepInput) item %s not found in collection", id)
	}
	return
}

func (c WorkflowContext) GetCommandLineTool(id string) (obj *CommandLineTool, err error) {
	obj, ok := c.CommandLineTools[id]
	if !ok {
		err = fmt.Errorf("(GetCommandLineTool) item %s not found in collection", id)
	}
	return
}

func (c WorkflowContext) GetExpressionTool(id string) (obj *ExpressionTool, err error) {
	obj, ok := c.ExpressionTools[id]
	if !ok {
		err = fmt.Errorf("(GetExpressionTool) item %s not found in collection", id)
	}
	return
}

func (c WorkflowContext) GetWorkflow(id string) (obj *Workflow, err error) {
	obj, ok := c.Workflows[id]
	if !ok {
		err = fmt.Errorf("(GetWorkflow) item %s not found in collection", id)
	}
	return
}

func NewWorkflowContext() (context *WorkflowContext) {
	context = &WorkflowContext{}

	context.Workflows = make(map[string]*Workflow)
	context.WorkflowStepInputs = make(map[string]*WorkflowStepInput)
	context.CommandLineTools = make(map[string]*CommandLineTool)
	context.ExpressionTools = make(map[string]*ExpressionTool)
	context.Files = make(map[string]*File)
	context.Strings = make(map[string]*String)
	context.Ints = make(map[string]*Int)
	context.Booleans = make(map[string]*Boolean)
	context.All = make(map[string]CWL_object)
	context.Schemata = make(map[string]CWLType_Type)
	return
}
