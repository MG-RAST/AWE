package cwl

import (
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/MG-RAST/AWE/lib/logger"
	rwmutex "github.com/MG-RAST/go-rwmutex"
	"github.com/davecgh/go-spew/spew"
)

// WorkflowContext global object for each job submission
type WorkflowContext struct {
	rwmutex.RWMutex
	Root GraphDocument `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // fields: CwlVersion, Base, Graph, Namespaces, Schemas (all interface-based !)
	Path string
	//Namespaces   map[string]string
	//CWLVersion
	//CwlVersion CWLVersion    `yaml:"cwl_version"  json:"cwl_version" bson:"cwl_version" mapstructure:"cwl_version"`
	//CWL_graph  []interface{} `yaml:"cwl_graph"  json:"cwl_graph" bson:"cwl_graph" mapstructure:"cwl_graph"`
	// old ParsingContext
	IfObjects map[string]interface{} `yaml:"-"  json:"-" bson:"-" mapstructure:"-"` // graph objects
	Objects   map[string]CWLObject   `yaml:"-"  json:"-" bson:"-" mapstructure:"-"` // graph objects , stores all objects (replaces All ???)

	FilesLoaded map[string]bool `yaml:"-"  json:"-" bson:"-" mapstructure:"-"`
	//Workflows          map[string]*Workflow          `yaml:"-"  json:"-" bson:"-" mapstructure:"-"`
	//InputParameter     map[string]*InputParameter    `yaml:"-"  json:"-" bson:"-" mapstructure:"-"` // WorkflowInput
	//WorkflowStepInputs map[string]*WorkflowStepInput `yaml:"-"  json:"-" bson:"-" mapstructure:"-"`
	//WorkflowStepInstance map[string]*WorkflowStep `yaml:"-"  json:"-" bson:"-" mapstructure:"-"`
	//CommandLineTools   map[string]*CommandLineTool   `yaml:"-"  json:"-" bson:"-" mapstructure:"-"`
	//ExpressionTools    map[string]*ExpressionTool    `yaml:"-"  json:"-" bson:"-" mapstructure:"-"`
	//Files              map[string]*File              `yaml:"-"  json:"-" bson:"-" mapstructure:"-"`
	//Strings            map[string]*String            `yaml:"-"  json:"-" bson:"-" mapstructure:"-"`
	//Ints               map[string]*Int               `yaml:"-"  json:"-" bson:"-" mapstructure:"-"`
	//Booleans           map[string]*Boolean           `yaml:"-"  json:"-" bson:"-" mapstructure:"-"`
	//All map[string]CWLObject `yaml:"-"  json:"-" bson:"-" mapstructure:"-"` // everything goes in here

	WorkflowCount int `yaml:"-"  json:"-" bson:"-" mapstructure:"-"`
	//Job_input          *Job_document
	//Job_input_map *JobDocMap `yaml:"-"  json:"-" bson:"-" mapstructure:"-"`

	Schemata    map[string]CWLType_Type `yaml:"-"  json:"-" bson:"-" mapstructure:"-"`
	Initialized bool                    `yaml:"-"  json:"-" bson:"-" mapstructure:"-"`
	Initialzing bool                    `yaml:"-"  json:"-" bson:"-" mapstructure:"-"` // collect objects in ths phase

	Name string `yaml:"-"  json:"-" bson:"-" mapstructure:"-"`
}

// NewWorkflowContext _
func NewWorkflowContext() (context *WorkflowContext) {

	logger.Debug(3, "(NewWorkflowContext) starting")

	context = &WorkflowContext{}
	context.Name = "George"
	return
}

// InitBasic _
func (context *WorkflowContext) InitBasic() {

	context.RWMutex.Init("context")

	if context.IfObjects == nil {
		context.IfObjects = make(map[string]interface{})
	}

	if context.Objects == nil {
		context.Objects = make(map[string]CWLObject)
	}

	// if context.All == nil {
	// 	context.All = make(map[string]CWLObject)
	// }

	//if context.WorkflowStepInstance == nil {
	//	context.WorkflowStepInstance = make(map[string]*WorkflowStep)
	//}

	if context.Schemata == nil {
		context.Schemata = make(map[string]CWLType_Type)
	}

	context.WorkflowCount = 0
	return
}

// Init search for #entrypoint and create objects recursively
func (context *WorkflowContext) Init(entrypoint string) (err error) {

	logger.Debug(3, "(WorkflowContext/Init) start")
	if context.Initialized == true {
		err = fmt.Errorf("(WorkflowContext/Init) already initialized")
		return
	}

	context.InitBasic()

	if context.Root.CwlVersion == "" {
		err = fmt.Errorf("(WorkflowContext/Init) context.Root.CwlVersion ==nil")
		return
	}

	graph := context.Root.Graph

	if len(graph) == 0 {
		err = fmt.Errorf("(WorkflowContext/Init) len(graph) == 0")
		return
	}

	logger.Debug(3, "(WorkflowContext/Init) len(graph): %d", len(graph))

	// put interface objetcs into map: populate context.If_objects
	for i, _ := range graph {

		//fmt.Printf("graph element type: %s\n", reflect.TypeOf(graph[i]))
		//spew.Dump(graph[i])

		if graph[i] == nil {
			err = fmt.Errorf("(WorkflowContext/Init) graph[i] empty array element")
			return
		}

		var id string
		id, err = GetID(graph[i])
		if err != nil {
			fmt.Println("(WorkflowContext/Init) object without id:")
			spew.Dump(graph[i])
			return
		}
		//fmt.Printf("id=\"%s\"\n", id)

		if !strings.HasPrefix(id, "#") {
			id = "#" + id

			// fix id in object
			var object interface{}

			object, err = MakeStringMap(graph[i], nil)
			if err != nil {
				return
			}

			var objectMap map[string]interface{}

			var ok bool
			objectMap, ok = object.(map[string]interface{})
			if ok {
				objectMap["id"] = id
				graph[i] = objectMap
			} else {
				err = fmt.Errorf("(WorkflowContext/Init) could nort parse object %s  (%s)", id, reflect.TypeOf(graph[i]))
				return
			}
			//fmt.Printf("new id=\"%s\"\n", id)
			//spew.Dump(graph[i])
		}

		context.IfObjects[id] = graph[i]

	}

	if entrypoint == "" { // for worker
		return
	}

	logger.Debug(3, "(WorkflowContext/Init) len(context.IfObjects): %d", len(context.IfObjects))

	entrypointIf, hasEntrypointObject := context.IfObjects[entrypoint] // e.g. #entrypoint
	if !hasEntrypointObject {

		if len(context.IfObjects) == 1 {
			for key, value := range context.IfObjects {
				entrypoint = key
				entrypointIf = value
			}
		}

		if entrypoint == "" {
			var keys string
			for key := range context.IfObjects {
				keys += "," + key
			}
			err = fmt.Errorf("(WorkflowContext/Init) entrypoint %s not found in graph (found %s)", entrypoint, keys)
			return
		}
	}

	if entrypointIf == nil {
		var keys string
		for key := range context.IfObjects {
			keys += "," + key
		}
		err = fmt.Errorf("(WorkflowContext/Init) entrypoint %s not found in graph ?? (found %s)", entrypoint, keys)
		return
	}

	// start with #entrypoint
	// recursivly add objects to context
	context.Initialzing = true
	var object CWLObject
	var schemataNew []CWLType_Type
	object, schemataNew, err = NewCWLObject(entrypointIf, "", entrypoint, nil, context)
	if err != nil {
		fmt.Printf("(WorkflowContext/Init) entrypointIf")
		spew.Dump(entrypointIf)
		err = fmt.Errorf("(WorkflowContext/Init) A NewCWLObject returned %s", err.Error())
		return
	}
	context.Initialzing = false
	context.Objects[entrypoint] = object

	err = context.AddSchemata(schemataNew, true)
	if err != nil {
		err = fmt.Errorf("(WorkflowContext/Init) context.AddSchemata returned %s", err.Error())
		return
	}
	//for i, _ := range schemataNew {
	//	schemata = append(schemata, schemataNew[i])
	//}
	//fmt.Println("context.All")
	//for key, _ := range context.All {
	//	fmt.Printf("context.All: %s\n", key)
	//}
	//panic("done")

	context.Root.Graph = nil
	context.Root.Graph = []interface{}{}
	for key, value := range context.Objects {
		logger.Debug(3, "(WorkflowContext/Init) adding %s to context.GraphDocument.Graph", key)
		// err = context.Add(key, value, "WorkflowContext/Init")
		// if err != nil {
		// 	err = fmt.Errorf("(WorkflowContext/Init) context.Add( returned %s", err.Error())
		// 	return
		// }

		context.Root.Graph = append(context.Root.Graph, value)
	}
	//fmt.Println("(WorkflowContext/Init) context.Objects: ")
	//spew.Dump(context.Objects)

	context.Initialized = true
	return
}

// Evaluate _
func (context *WorkflowContext) Evaluate(raw string) (parsed string) {

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
			value, err := context.GetString(string(key))

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

// AddSchemata _
func (context *WorkflowContext) AddSchemata(obj []CWLType_Type, writeLock bool) (err error) {
	//fmt.Printf("(AddSchemata)\n")
	if writeLock {
		err = context.LockNamed("AddSchemata")
		if err != nil {
			return
		}
		defer context.Unlock()
	}

	if context.Schemata == nil {
		context.Schemata = make(map[string]CWLType_Type)
	}

	for i := range obj {
		id := obj[i].GetID()
		if id == "" {
			err = fmt.Errorf("id empty")
			return
		}

		idArray := strings.Split(id, "#")
		id = idArray[len(idArray)-1]
		//fmt.Printf("Adding %s\n", id)

		_, ok := context.Schemata[id]
		if ok {
			return
		}

		context.Schemata[id] = obj[i]
	}
	return
}

// GetSchemata _
func (context *WorkflowContext) GetSchemata() (obj []CWLType_Type, err error) {
	obj = []CWLType_Type{}
	for _, schema := range context.Schemata {
		obj = append(obj, schema)
	}
	return
}

// AddArray _
func (context *WorkflowContext) AddArray(objectArray []NamedCWLObject) (err error) {

	for i := range objectArray {
		pair := objectArray[i]

		err = context.AddObject(pair.ID, pair.Value, "AddArray")
		if err != nil {
			return
		}

	}

	return

}

// AddObject _
func (context *WorkflowContext) AddObject(id string, obj CWLObject, caller string) (err error) {

	if id == "" {
		// anonymous objects are not stored
		return
	}

	if !strings.HasPrefix(id, "#") {
		err = fmt.Errorf("(WorkflowContext/Add) id %s is not absolute", id)
		return
	}

	logger.Debug(3, "(WorkflowContext/Add) Adding object %s to collection (type: %s, caller: %s)", id, reflect.TypeOf(obj), caller)

	if context.Objects == nil {
		context.Objects = make(map[string]CWLObject)
	}

	_, ok := context.Objects[id]
	if ok {
		err = fmt.Errorf("(WorkflowContext/Add) Object %s already in collection (caller: %s)", id, caller)
		return
	}

	switch obj.(type) {
	case *Workflow:
		//fmt.Printf("(c.Objects) c.WorkflowCount: %d\n", c.WorkflowCount)
		context.WorkflowCount++
		//fmt.Printf("(c.Objects) c.WorkflowCount: %d\n", c.WorkflowCount)
		msg := fmt.Sprintf("(WorkflowContext/Add) new WorkflowCount: %d (context: %p, caller: %s, name: %s)", context.WorkflowCount, &context, caller, context.Name)
		logger.Debug(3, msg)
		//fmt.Printf("(c.Objects) msg: %s\n", msg)
		//for i, _ := range c.Objects {
		//	fmt.Println(i)
		//}

	//	c.Workflows[id] = obj.(*Workflow)
	case *WorkflowStepInput:
		objReal, ok := obj.(*WorkflowStepInput)
		if !ok {
			err = fmt.Errorf("could not make WorkflowStepInput type assertion")
			return
		}
		context.Objects[id] = objReal
	case *CommandLineTool:
		objReal, ok := obj.(*CommandLineTool)
		if !ok {
			err = fmt.Errorf("could not make CommandLineTool type assertion")
			return
		}
		context.Objects[id] = objReal
	case *ExpressionTool:
		objReal, ok := obj.(*ExpressionTool)
		if !ok {
			err = fmt.Errorf("could not make ExpressionTool type assertion")
			return
		}
		context.Objects[id] = objReal
	case *File:
		objReal, ok := obj.(*File)
		if !ok {
			err = fmt.Errorf("could not make File type assertion")
			return
		}
		context.Objects[id] = objReal
	case *String:
		objReal, ok := obj.(*String)
		if !ok {
			err = fmt.Errorf("could not make String type assertion")
			return
		}
		context.Objects[id] = objReal
	case *Boolean:
		objReal, ok := obj.(*Boolean)
		if !ok {
			err = fmt.Errorf("could not make Boolean type assertion")
			return
		}
		context.Objects[id] = objReal
	case *Int:
		objInt, ok := obj.(*Int)
		if !ok {
			err = fmt.Errorf("could not make Int type assertion")
			return
		}
		context.Objects[id] = objInt
	default:
		logger.Debug(3, "adding type %s to WorkflowContext.Objects", reflect.TypeOf(obj))
	}

	context.Objects[id] = obj
	//fmt.Printf("(c.All) after insertion of %s (caller: %s)\n", id, caller)
	//for i, _ := range c.All {
	//	fmt.Println(i)
	//}
	return
}

// Get _
func (context *WorkflowContext) Get(id string, doReadLock bool) (obj CWLObject, ok bool, err error) {

	if doReadLock {
		var readLock rwmutex.ReadLock
		readLock, err = context.RLockNamed("WorkflowContext/Get")
		if err != nil {
			return
		}
		defer context.RUnlockNamed(readLock)
	}

	obj, ok = context.Objects[id]
	if !ok {
		logger.Debug(3, "(WorkflowContext/Get) did not find: %s", id)
		for k := range context.Objects {
			logger.Debug(3, "(WorkflowContext/Get) available: %s", k)
		}
		//err = fmt.Errorf("(All) item %s not found in collection", id)
	}

	return
}

// GetType _
func (context *WorkflowContext) GetType(id string) (objType string, err error) {
	doReadLock := true
	if doReadLock {
		readLock, xerr := context.RLockNamed("WorkflowContext/Get")
		if xerr != nil {
			err = xerr
			return
		}
		defer context.RUnlockNamed(readLock)
	}
	var ok bool
	var obj CWLObject
	obj, ok = context.Objects[id]
	if !ok {
		err = fmt.Errorf("(GetCWLTypeType) Object %s not found in All", id)
		return
	}

	objType = fmt.Sprintf("%s", reflect.TypeOf(obj))

	return

}

// func (c *WorkflowContext) GetCWLType(id string) (obj CWLType, err error) {
// 	var ok bool
// 	obj, ok = c.Files[id]
// 	if ok {
// 		return
// 	}
// 	obj, ok = c.Strings[id]
// 	if ok {
// 		return
// 	}

// 	obj, ok = c.Ints[id]
// 	if ok {
// 		return
// 	}
// 	obj, ok = c.Booleans[id]
// 	if ok {
// 		return
// 	}

// 	err = fmt.Errorf("(GetType) %s not found", id)
// 	return

// }

// GetFile _
func (context *WorkflowContext) GetFile(id string) (obj *File, err error) {
	var objGeneric CWLObject
	var ok bool
	objGeneric, ok, err = context.Get(id, true)
	if err != nil {
		err = fmt.Errorf("(GetFile) error getting item %s: %s", id, err.Error())
		return
	}
	if !ok {
		err = fmt.Errorf("(GetFile) item %s not found in collection: %s", id, err.Error())
		return
	}

	obj, ok = objGeneric.(*File)
	if !ok {
		err = fmt.Errorf("(GetFile) Item %s has wrong type: %s", id, reflect.TypeOf(objGeneric))
	}
	return
}

// GetString _
func (context *WorkflowContext) GetString(id string) (obj *String, err error) {
	var objGeneric CWLObject
	var ok bool
	objGeneric, ok, err = context.Get(id, true)
	if err != nil {
		err = fmt.Errorf("(GetString) error getting item %s: %s", id, err.Error())
		return
	}
	if !ok {
		err = fmt.Errorf("(GetString) item %s not found in collection: %s", id, err.Error())
		return
	}

	obj, ok = objGeneric.(*String)
	if !ok {
		err = fmt.Errorf("(GetString) Item %s has wrong type: %s", id, reflect.TypeOf(objGeneric))
	}
	return
}

// GetInt _
func (context *WorkflowContext) GetInt(id string) (obj *Int, err error) {
	var objGeneric CWLObject
	var ok bool
	objGeneric, ok, err = context.Get(id, true)
	if err != nil {
		err = fmt.Errorf("(GetInt) error getting item %s: %s", id, err.Error())
		return
	}
	if !ok {
		err = fmt.Errorf("(GetInt) item %s not found in collection: %s", id, err.Error())
		return
	}

	obj, ok = objGeneric.(*Int)
	if !ok {
		err = fmt.Errorf("(GetInt) Item %s has wrong type: %s", id, reflect.TypeOf(objGeneric))
	}
	return
}

// GetWorkflowStepInput _
func (context *WorkflowContext) GetWorkflowStepInputDeprecated(id string) (obj *WorkflowStepInput, err error) {
	var objGeneric CWLObject
	var ok bool
	objGeneric, ok, err = context.Get(id, true)
	if err != nil {
		err = fmt.Errorf("(GetWorkflowStepInput) error getting item %s: %s", id, err.Error())
		return
	}
	if !ok {
		err = fmt.Errorf("(GetWorkflowStepInput) item %s not found in collection: %s", id, err.Error())
		return
	}

	obj, ok = objGeneric.(*WorkflowStepInput)
	if !ok {
		err = fmt.Errorf("(GetWorkflowStepInput) Item %s has wrong type: %s", id, reflect.TypeOf(objGeneric))
	}
	return
}

// GetCommandLineTool _
func (context *WorkflowContext) GetCommandLineTool(id string) (obj *CommandLineTool, err error) {
	var objGeneric CWLObject
	var ok bool
	objGeneric, ok, err = context.Get(id, true)
	if err != nil {
		err = fmt.Errorf("(GetCommandLineTool) error getting item %s: %s", id, err.Error())
		return
	}
	if !ok {
		err = fmt.Errorf("(GetCommandLineTool) item %s not found in collection", id)
		return
	}

	obj, ok = objGeneric.(*CommandLineTool)
	if !ok {
		err = fmt.Errorf("(GetCommandLineTool) Item %s has wrong type: %s", id, reflect.TypeOf(objGeneric))
	}
	return
}

// GetExpressionTool _
func (context *WorkflowContext) GetExpressionTool(id string) (obj *ExpressionTool, err error) {
	var objGeneric CWLObject
	var ok bool
	objGeneric, ok, err = context.Get(id, true)
	if err != nil {
		err = fmt.Errorf("(GetExpressionTool) error getting item %s: %s", id, err.Error())
		return
	}
	if !ok {
		err = fmt.Errorf("(GetExpressionTool) item %s not found in collection", id)
		return
	}

	obj, ok = objGeneric.(*ExpressionTool)
	if !ok {
		err = fmt.Errorf("(GetExpressionTool) Item %s has wrong type: %s", id, reflect.TypeOf(objGeneric))
	}
	return
}

// GetWorkflow _
func (context *WorkflowContext) GetWorkflow(id string) (obj *Workflow, err error) {
	var objGeneric CWLObject
	var ok bool
	objGeneric, ok, err = context.Get(id, true)
	if err != nil {
		err = fmt.Errorf("(GetWorkflow) error getting item %s: %s", id, err.Error())
		return
	}
	if !ok {

		keys := ""
		for key := range context.Objects {
			keys += "," + key
		}

		err = fmt.Errorf("(GetWorkflow) item %s not found in collection (found: %s)", id, keys)
		return
	}

	obj, ok = objGeneric.(*Workflow)
	if !ok {
		err = fmt.Errorf("(GetWorkflow) Item %s has wrong type: %s", id, reflect.TypeOf(objGeneric))
	}
	return
}
