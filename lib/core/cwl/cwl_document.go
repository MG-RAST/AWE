package cwl

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
)

// this is used by YAML or JSON library for inital parsing
type CWL_document struct {
	CwlVersion CWLVersion        `yaml:"cwlVersion,omitempty" json:"cwlVersion,omitempty" bson:"cwlVersion,omitempty" mapstructure:"cwlVersion,omitempty"`
	Base       interface{}       `yaml:"$base,omitempty" json:"$base,omitempty" bson:"base,omitempty" mapstructure:"$base,omitempty"`
	Graph      []interface{}     `yaml:"$graph" json:"$graph" bson:"graph" mapstructure:"$graph"` // can only be used for reading, yaml has problems writing interface objetcs
	Namespaces map[string]string `yaml:"$namespaces,omitempty" json:"$namespaces,omitempty" bson:"namespaces,omitempty" mapstructure:"$namespaces,omitempty"`
	Schemas    []interface{}     `yaml:"$schemas,omitempty" json:"$schemas,omitempty" bson:"schemas,omitempty" mapstructure:"$schemas,omitempty"`
}

func Parse_cwl_graph_document(yaml_str string, context *WorkflowContext) (object_array []Named_CWL_object, schemata []CWLType_Type, schemas []interface{}, err error) {

	cwl_gen := CWL_document{}

	yaml_byte := []byte(yaml_str)
	err = Unmarshal(&yaml_byte, &cwl_gen)
	if err != nil {
		logger.Debug(1, "CWL unmarshal error")
		err = fmt.Errorf("(Parse_cwl_graph_document) Unmarshal returned: " + err.Error())
		return
	}
	//fmt.Println("-------------- yaml_str")
	//fmt.Println(yaml_str)
	//fmt.Println("-------------- raw CWL")
	//spew.Dump(cwl_gen)
	//fmt.Println("-------------- Start parsing")

	if len(cwl_gen.Graph) == 0 {
		err = fmt.Errorf("(Parse_cwl_graph_document) len(cwl_gen.Graph) == 0")
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

	//context.Graph = cwl_gen.Graph
	context.CWL_document = cwl_gen
	context.CwlVersion = cwl_gen.CwlVersion

	if cwl_gen.Namespaces != nil {
		context.Namespaces = cwl_gen.Namespaces
	}

	if cwl_gen.Schemas != nil {
		schemas = cwl_gen.Schemas
	}

	// iterate over Graph

	// try to find CWL version!
	if context.CwlVersion == "" {
		for _, elem := range cwl_gen.Graph {
			elem_map, ok := elem.(map[string]interface{})
			if ok {
				cwl_version_if, has_version := elem_map["cwlVersion"]
				if has_version {

					var cwl_version_str string
					cwl_version_str, ok = cwl_version_if.(string)
					if !ok {
						err = fmt.Errorf("(Parse_cwl_graph_document) Could not read CWLVersion (%s)", reflect.TypeOf(cwl_version_if))
						return
					}
					context.CwlVersion = CWLVersion(cwl_version_str)
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

	err = context.Init("#main")
	if err != nil {
		err = fmt.Errorf("(Parse_cwl_graph_document) context.FillMaps returned: %s", err.Error())
		return
	}

	for id, object := range context.Objects {
		named_obj := NewNamed_CWL_object(id, object)
		object_array = append(object_array, named_obj)
	}

	// object_array []Named_CWL_object
	//fmt.Println("############# object_array:")
	//spew.Dump(object_array)

	return
}

func Parse_cwl_simple_document(yaml_str string, context *WorkflowContext) (object_array []Named_CWL_object, schemata []CWLType_Type, schemas []interface{}, err error) {
	// Here I expect a single object, Workflow or CommandLIneTool
	//fmt.Printf("-------------- yaml_str: %s\n", yaml_str)

	var object_if map[string]interface{}

	yaml_byte := []byte(yaml_str)
	err = Unmarshal(&yaml_byte, &object_if)
	if err != nil {
		//logger.Debug(1, "CWL unmarshal error")
		err = fmt.Errorf("(Parse_cwl_simple_document) Unmarshal returns: %s", err.Error())
		return
	}
	if context.CwlVersion == "" {

		cwl_version_if, has_version := object_if["cwlVersion"]
		if has_version {

			cwl_version_str, ok := cwl_version_if.(string)
			if ok {
				context.CwlVersion = NewCWLVersion(cwl_version_str)
			} else {
				err = fmt.Errorf("(Parse_cwl_simple_document) version not string (type: %s)", reflect.TypeOf(cwl_version_if))
				return
			}
		} else {
			spew.Dump(object_if)
			err = fmt.Errorf("(Parse_cwl_simple_document) no version found")
			return
		}
	}
	//fmt.Println("object_if:")
	//spew.Dump(object_if)
	var ok bool
	var namespaces_if interface{}
	namespaces_if, ok = object_if["$namespaces"]
	if ok {
		var namespaces_map map[string]interface{}
		namespaces_map, ok = namespaces_if.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(Parse_cwl_simple_document) namespaces_if error (type: %s)", reflect.TypeOf(namespaces_if))
			return
		}
		context.Namespaces = make(map[string]string)

		for key, value := range namespaces_map {

			switch value := value.(type) {
			case string:
				context.Namespaces[key] = value
			default:
				err = fmt.Errorf("(Parse_cwl_simple_document) value is not string (%s)", reflect.TypeOf(value))
				return
			}

		}

	} else {
		context.Namespaces = nil
		//fmt.Println("no namespaces")
	}

	var schemas_if interface{}
	schemas_if, ok = object_if["$schemas"]
	if ok {

		schemas, ok = schemas_if.([]interface{})
		if !ok {
			err = fmt.Errorf("(Parse_cwl_simple_document) could not parse $schemas (%s)", reflect.TypeOf(schemas_if))
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

	var this_id string
	this_id, err = GetId(object_if)
	if err != nil {
		err = fmt.Errorf("(Parse_cwl_simple_document) GetId returns %s", err.Error())
		return
	}
	//fmt.Printf("this_id: %s\n", this_id)

	var object CWL_object
	var schemata_new []CWLType_Type
	object, schemata_new, err = New_CWL_object(object_if, nil, context)
	if err != nil {
		err = fmt.Errorf("(Parse_cwl_simple_document) B New_CWL_object returns %s", err.Error())
		return
	}

	//fmt.Println("-------------- raw CWL")
	//spew.Dump(commandlinetool_if)
	//fmt.Println("-------------- Start parsing")

	//var commandlinetool *CommandLineTool
	//var schemata_new []CWLType_Type
	//commandlinetool, schemata_new, err = NewCommandLineTool(commandlinetool_if)
	//if err != nil {
	//	err = fmt.Errorf("(Parse_cwl_document) NewCommandLineTool returned: %s", err.Error())
	//	return
	//}

	switch object.(type) {
	case *Workflow:
		this_workflow, _ := object.(*Workflow)
		context.CwlVersion = this_workflow.CwlVersion
	case *CommandLineTool:
		this_clt, _ := object.(*CommandLineTool)
		context.CwlVersion = this_clt.CwlVersion
	case *ExpressionTool:
		this_et, _ := object.(*ExpressionTool)
		context.CwlVersion = this_et.CwlVersion
	default:

		err = fmt.Errorf("(Parse_cwl_simple_document) type unkown: %s", reflect.TypeOf(object))
		return
	}

	named_obj := NewNamed_CWL_object(this_id, object)
	//named_obj := NewNamed_CWL_object(commandlinetool.Id, commandlinetool)

	//cwl_version = commandlinetool.CwlVersion // TODO

	object_array = append(object_array, named_obj)
	for i, _ := range schemata_new {
		schemata = append(schemata, schemata_new[i])
	}
	return
}

func Parse_cwl_document(yaml_str string, inputfile_path string) (object_array []Named_CWL_object, schemata []CWLType_Type, context *WorkflowContext, schemas []interface{}, err error) {
	//fmt.Printf("(Parse_cwl_document) starting\n")

	context = NewWorkflowContext()
	context.Path = inputfile_path
	context.InitBasic()
	graph_pos := strings.Index(yaml_str, "$graph")

	//yaml_str = strings.Replace(yaml_str, "$namespaces", "namespaces", -1)

	if graph_pos != -1 {
		// *** graph file ***
		//yaml_str = strings.Replace(yaml_str, "$graph", "graph", -1) // remove dollar sign
		logger.Debug(3, "(Parse_cwl_document) graph document")
		object_array, schemata, schemas, err = Parse_cwl_graph_document(yaml_str, context)
		if err != nil {
			err = fmt.Errorf("(Parse_cwl_document) Parse_cwl_graph_document returned: %s", err.Error())
			return
		}

	} else {
		logger.Debug(3, "(Parse_cwl_document) simple document")
		object_array, schemata, schemas, err = Parse_cwl_simple_document(yaml_str, context)
		if err != nil {
			err = fmt.Errorf("(Parse_cwl_document) Parse_cwl_simple_document returned: %s", err.Error())
			return
		}
	}
	if len(object_array) == 0 {
		err = fmt.Errorf("(Parse_cwl_document) len(object_array) == 0 (graph_pos: %d)", graph_pos)
		return
	}

	return
}
