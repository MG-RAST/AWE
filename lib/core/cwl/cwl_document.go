package cwl

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
)

// GraphDocument this is used by YAML or JSON library for inital parsing
type GraphDocument struct {
	CwlVersion CWLVersion        `yaml:"cwlVersion,omitempty" json:"cwlVersion,omitempty" bson:"cwlVersion,omitempty" mapstructure:"cwlVersion,omitempty"`
	Base       interface{}       `yaml:"$base,omitempty" json:"$base,omitempty" bson:"base,omitempty" mapstructure:"$base,omitempty"`
	Graph      []interface{}     `yaml:"$graph" json:"$graph" bson:"graph" mapstructure:"$graph"` // can only be used for reading, yaml has problems writing interface objetcs
	Namespaces map[string]string `yaml:"$namespaces,omitempty" json:"$namespaces,omitempty" bson:"namespaces,omitempty" mapstructure:"$namespaces,omitempty"`
	Schemas    []interface{}     `yaml:"$schemas,omitempty" json:"$schemas,omitempty" bson:"schemas,omitempty" mapstructure:"$schemas,omitempty"`
}

// ParseCWLGraphDocument _
// entrypoint defauyts to #main but can anything else...
func ParseCWLGraphDocument(yamlStr string, entrypoint string, context *WorkflowContext) (objectArray []NamedCWLObject, schemata []CWLType_Type, schemas []interface{}, err error) {

	cwlGen := GraphDocument{}

	yamlByte := []byte(yamlStr)
	err = Unmarshal(&yamlByte, &cwlGen)
	if err != nil {
		logger.Debug(1, "CWL unmarshal error")
		err = fmt.Errorf("(Parse_cwl_graph_document) Unmarshal returned: " + err.Error())
		return
	}
	//fmt.Println("-------------- yamlStr")
	//fmt.Println(yamlStr)
	//fmt.Println("-------------- raw CWL")
	//spew.Dump(cwl_gen)
	//fmt.Println("-------------- Start parsing")

	if len(cwlGen.Graph) == 0 {
		err = fmt.Errorf("(Parse_cwl_graph_document) len(cwlGen.Graph) == 0")
		return
	}

	// resolve $import
	//var new_obj []interface{}
	//err = Resolve_ImportsArray(cwl_gen.Graph, context)
	//if err != nil {
	//	err = fmt.Errorf("(Parse_cwl_graph_document) Resolve_Imports returned: " + err.Error())
	//	return
	//}

	//fmt.Println("############# cwl_gen.Graph:")
	//spew.Dump(cwl_gen.Graph)
	//panic("done")
	//if new_obj != nil {
	//	err = fmt.Errorf("(Parse_cwl_graph_document) $import in $graph not supported yet")
	//	return
	//}

	//context.Graph = cwlGen.Graph
	context.GraphDocument = cwlGen
	context.CwlVersion = cwlGen.CwlVersion

	if cwlGen.Namespaces != nil {
		context.Namespaces = cwlGen.Namespaces
	}

	if cwlGen.Schemas != nil {
		schemas = cwlGen.Schemas
	}

	// iterate over Graph

	// try to find CWL version!
	if context.CwlVersion == "" {
		for _, elem := range cwlGen.Graph {
			elemMap, ok := elem.(map[string]interface{})
			if ok {
				cwlVersionIf, hasVersion := elemMap["cwlVersion"]
				if hasVersion {

					var cwlVersionStr string
					cwlVersionStr, ok = cwlVersionIf.(string)
					if !ok {
						err = fmt.Errorf("(Parse_cwl_graph_document) Could not read CWLVersion (%s)", reflect.TypeOf(cwlVersionIf))
						return
					}
					context.CwlVersion = CWLVersion(cwlVersionStr)
					break
				}

			}
		}

	}

	if context.CwlVersion == "" {
		// see raw
		err = fmt.Errorf("(Parse_cwl_graph_document) cwl_version empty")
		return
	}

	//fmt.Println("-------------- A Parse_cwl_document")
	if entrypoint == "" {
		entrypoint = "#main"
	}

	err = context.Init(entrypoint)
	if err != nil {
		err = fmt.Errorf("(Parse_cwl_graph_document) context.Init returned: %s", err.Error())
		return
	}

	for id, object := range context.Objects {
		namedObj := NewNamedCWLObject(id, object)
		objectArray = append(objectArray, namedObj)
	}

	// object_array []NamedCWLObject
	//fmt.Println("############# object_array:")
	//spew.Dump(object_array)

	return
}

// ParseCWLSimpleDocument _
func ParseCWLSimpleDocument(yamlStr string, basename string, context *WorkflowContext) (objectArray []NamedCWLObject, schemata []CWLType_Type, schemas []interface{}, err error) {
	// Here I expect a single object, Workflow or CommandLIneTool
	//fmt.Printf("-------------- yaml_str: %s\n", yaml_str)

	var objectIf map[string]interface{}

	yamlByte := []byte(yamlStr)
	err = Unmarshal(&yamlByte, &objectIf)
	if err != nil {
		//logger.Debug(1, "CWL unmarshal error")
		err = fmt.Errorf("(ParseCWLSimpleDocument) Unmarshal returns: %s", err.Error())
		return
	}
	if context.CwlVersion == "" {

		cwlVersionIf, hasVersion := objectIf["cwlVersion"]
		if hasVersion {

			cwlVersionStr, ok := cwlVersionIf.(string)
			if ok {
				context.CwlVersion = NewCWLVersion(cwlVersionStr)
			} else {
				err = fmt.Errorf("(ParseCWLSimpleDocument) version not string (type: %s)", reflect.TypeOf(cwlVersionIf))
				return
			}
		} else {
			spew.Dump(objectIf)
			err = fmt.Errorf("(ParseCWLSimpleDocument) no version found")
			return
		}
	}

	if objectIf == nil {
		err = fmt.Errorf("(ParseCWLSimpleDocument) objectIf == nil")
		return
	}

	fmt.Println("---------- A\n")

	//fmt.Println("object_if:")
	//spew.Dump(object_if)
	var ok bool
	var namespacesIf interface{}
	namespacesIf, ok = objectIf["$namespaces"]
	if ok {
		var namespacesMap map[string]interface{}
		namespacesMap, ok = namespacesIf.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(ParseCWLSimpleDocument) namespaces_if error (type: %s)", reflect.TypeOf(namespacesIf))
			return
		}
		context.Namespaces = make(map[string]string)

		for key, value := range namespacesMap {

			switch value := value.(type) {
			case string:
				context.Namespaces[key] = value
			default:
				err = fmt.Errorf("(ParseCWLSimpleDocument) value is not string (%s)", reflect.TypeOf(value))
				return
			}

		}

	} else {
		context.Namespaces = nil
		//fmt.Println("no namespaces")
	}
	fmt.Println("---------- B\n")
	var schemasIf interface{}
	schemasIf, ok = objectIf["$schemas"]
	if ok {

		schemas, ok = schemasIf.([]interface{})
		if !ok {
			err = fmt.Errorf("(ParseCWLSimpleDocument) could not parse $schemas (%s)", reflect.TypeOf(schemasIf))
			return
		}

	}
	//var this_class string
	//this_class, err = GetClass(object_if)
	//if err != nil {
	//	err = fmt.Errorf("(Parse_cwl_document) GetClass returns %s", err.Error())
	//	return
	//}
	//fmt.Printf("this_class: %s\n", this_class)
	fmt.Println("---------- C\n")
	setObjectID := false

	var objectID string
	objectID, err = GetId(objectIf)
	if err != nil {
		//fmt.Printf("ParseCWLSimpleDocument: got no id\n")

		if basename == "" {
			err = fmt.Errorf("(ParseCWLSimpleDocument) B) GetId returns %s", err.Error())
			return
		}

		objectID = "#" + basename
		//fmt.Printf("ParseCWLSimpleDocument: using %s\n", basename)
		setObjectID = true
		//err = fmt.Errorf("(ParseCWLSimpleDocument) GetId returns %s", err.Error())
		//return
	}
	//fmt.Printf("objectID: %s\n", objectID)

	var object CWLObject
	var schemataNew []CWLType_Type
	object, schemataNew, err = NewCWLObject(objectIf, objectID, nil, context)
	if err != nil {
		err = fmt.Errorf("(ParseCWLSimpleDocument) B NewCWLObject returns %s", err.Error())
		return
	}

	//fmt.Println("-------------- raw CWL")
	//spew.Dump(commandlinetool_if)
	//fmt.Println("-------------- Start parsing")

	//var commandlinetool *CommandLineTool
	//var schemataNew []CWLType_Type
	//commandlinetool, schemataNew, err = NewCommandLineTool(commandlinetool_if)
	//if err != nil {
	//	err = fmt.Errorf("(Parse_cwl_document) NewCommandLineTool returned: %s", err.Error())
	//	return
	//}

	switch object.(type) {
	case *Workflow:
		thisWorkflow, _ := object.(*Workflow)
		if setObjectID {
			thisWorkflow.Id = objectID
		}
		context.CwlVersion = thisWorkflow.CwlVersion
	case *CommandLineTool:
		thisCLT, _ := object.(*CommandLineTool)
		if setObjectID {
			thisCLT, _ := object.(*CommandLineTool)
			thisCLT.Id = objectID
		}
		context.CwlVersion = thisCLT.CwlVersion
	case *ExpressionTool:
		thisET, _ := object.(*ExpressionTool)
		if setObjectID {
			thisET.Id = objectID
		}
		context.CwlVersion = thisET.CwlVersion
	default:

		err = fmt.Errorf("(ParseCWLSimpleDocument) type unkown: %s", reflect.TypeOf(object))
		return
	}

	namedObj := NewNamedCWLObject(objectID, object)
	//named_obj := NewNamedCWLObject(commandlinetool.Id, commandlinetool)

	//cwl_version = commandlinetool.CwlVersion // TODO

	objectArray = append(objectArray, namedObj)
	for i := range schemataNew {
		schemata = append(schemata, schemataNew[i])
	}
	return
}

// ParseCWLDocument _
// format: inputfilePath  / fileName # entrypoint , example: /path/tool.cwl#main
// a CWL document can be a graph document or a single object document
// an entrypoint can only be specified for a graph document
func ParseCWLDocument(yamlStr string, entrypoint string, inputfilePath string, fileName string) (objectArray []NamedCWLObject, schemata []CWLType_Type, context *WorkflowContext, schemas []interface{}, err error) {
	fmt.Printf("(Parse_cwl_document) starting\n")

	context = NewWorkflowContext()
	context.Path = inputfilePath
	context.InitBasic()
	graphPos := strings.Index(yamlStr, "$graph")

	//yamlStr = strings.Replace(yamlStr, "$namespaces", "namespaces", -1)
	//fmt.Println("yamlStr:")
	//fmt.Println(yamlStr)

	if graphPos != -1 {
		// *** graph file ***
		//yamlStr = strings.Replace(yamlStr, "$graph", "graph", -1) // remove dollar sign
		logger.Debug(3, "(Parse_cwl_document) graph document")
		fmt.Printf("(Parse_cwl_document) ParseCWLGraphDocument\n")
		objectArray, schemata, schemas, err = ParseCWLGraphDocument(yamlStr, entrypoint, context)
		if err != nil {
			err = fmt.Errorf("(Parse_cwl_document) Parse_cwl_graph_document returned: %s", err.Error())
			return
		}

	} else {
		logger.Debug(3, "(Parse_cwl_document) simple document")
		fmt.Printf("(Parse_cwl_document) ParseCWLSimpleDocument\n")
		objectArray, schemata, schemas, err = ParseCWLSimpleDocument(yamlStr, fileName, context)
		if err != nil {
			err = fmt.Errorf("(Parse_cwl_document) Parse_cwl_simple_document returned: %s", err.Error())
			return
		}
	}
	if len(objectArray) == 0 {
		err = fmt.Errorf("(Parse_cwl_document) len(objectArray) == 0 (graphPos: %d)", graphPos)
		return
	}
	fmt.Printf("(Parse_cwl_document) end\n")
	return
}
