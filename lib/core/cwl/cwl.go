package cwl

import (
	"encoding/json"
	"reflect"

	"fmt"

	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
	//"github.com/mitchellh/mapstructure"

	"gopkg.in/yaml.v2"
	//"io/ioutil"
	//"os"
	//"reflect"
	"strconv"
	"strings"
)

// this is used by YAML or JSON library for inital parsing
type CWL_document_generic struct {
	CwlVersion CWLVersion        `yaml:"cwlVersion"`
	Graph      []interface{}     `yaml:"graph"`
	Namespaces map[string]string `yaml:"namespaces"`
	//Graph      []CWL_object_generic `yaml:"graph"`
}

type ParsingContext struct {
	If_objects map[string]interface{}
	Objects    map[string]CWL_object
}

type CWL_object_generic map[string]interface{}

type CWLVersion string

type LinkMergeMethod string // merge_nested or merge_flattened

func Parse_cwl_document(yaml_str string) (object_array []Named_CWL_object, cwl_version CWLVersion, schemata []CWLType_Type, namespaces map[string]string, err error) {
	//fmt.Printf("(Parse_cwl_document) starting\n")
	graph_pos := strings.Index(yaml_str, "$graph")

	yaml_str = strings.Replace(yaml_str, "$namespaces", "namespaces", -1)

	if graph_pos != -1 {
		// *** graph file ***
		yaml_str = strings.Replace(yaml_str, "$graph", "graph", -1) // remove dollar sign

		cwl_gen := CWL_document_generic{}

		yaml_byte := []byte(yaml_str)
		err = Unmarshal(&yaml_byte, &cwl_gen)
		if err != nil {
			logger.Debug(1, "CWL unmarshal error")
			err = fmt.Errorf("(Parse_cwl_document) Unmarshal returned:" + err.Error())
			return
		}
		//fmt.Println("-------------- yaml_str")
		//fmt.Println(yaml_str)
		//fmt.Println("-------------- raw CWL")
		//spew.Dump(cwl_gen)
		//fmt.Println("-------------- Start parsing")

		cwl_version = cwl_gen.CwlVersion

		if cwl_gen.Namespaces != nil {
			namespaces = cwl_gen.Namespaces
		}
		// iterate over Graph

		// try to find CWL version!
		if cwl_version == "" {
			for _, elem := range cwl_gen.Graph {
				elem_map, ok := elem.(map[string]interface{})
				if ok {
					cwl_version_if, has_version := elem_map["cwlVersion"]
					if has_version {

						var cwl_version_str string
						cwl_version_str, ok = cwl_version_if.(string)
						if !ok {
							err = fmt.Errorf("(Parse_cwl_document) Could not read CWLVersion (%s)", reflect.TypeOf(cwl_version_if))
							return
						}
						cwl_version = CWLVersion(cwl_version_str)
						break
					}

				}
			}

		}

		if cwl_version == "" {
			// see raw
			err = fmt.Errorf("(Parse_cwl_document) cwl_version empty")
			return
		}

		//fmt.Println("-------------- A Parse_cwl_document")

		context := &ParsingContext{}
		context.If_objects = make(map[string]interface{})
		context.Objects = make(map[string]CWL_object)

		// turn array into map: populate context.If_objects
		for _, elem := range cwl_gen.Graph {

			var id string
			id, err = GetId(elem)
			if err != nil {
				fmt.Println("object without id:")
				spew.Dump(elem)
				return
			}
			//fmt.Printf("id=\"%s\\n", id)

			context.If_objects[id] = elem

			// var class string
			// class, err = GetClass(elem)
			// if err != nil {
			// 	fmt.Println("object without class:")
			// 	spew.Dump(elem)
			// 	return
			// }
			// _ = class
			_ = id
		}

		main_if, has_main := context.If_objects["#main"]
		if !has_main {
			var keys string
			for key, _ := range context.If_objects {
				keys += "," + key
			}
			err = fmt.Errorf("(Parse_cwl_document) #main not found in graph (found %s)", keys)
			return
		}

		// start with #main
		// recursivly add objects to context
		var object CWL_object
		var schemata_new []CWLType_Type
		object, schemata_new, err = New_CWL_object(main_if, cwl_version, nil, namespaces, context)
		if err != nil {
			err = fmt.Errorf("(Parse_cwl_document) A New_CWL_object returns %s", err.Error())
			return
		}
		context.Objects["#main"] = object

		for i, _ := range schemata_new {
			schemata = append(schemata, schemata_new[i])
		}

		for id, object := range context.Objects {
			named_obj := NewNamed_CWL_object(id, object)
			object_array = append(object_array, named_obj)
		}
	} else {

		// Here I expect a single object, Workflow or CommandLIneTool
		//fmt.Printf("-------------- yaml_str: %s\n", yaml_str)

		var object_if map[string]interface{}

		yaml_byte := []byte(yaml_str)
		err = Unmarshal(&yaml_byte, &object_if)
		if err != nil {
			//logger.Debug(1, "CWL unmarshal error")
			err = fmt.Errorf("(Parse_cwl_document) Unmarshal returns: %s", err.Error())
			return
		}
		//fmt.Println("object_if:")
		//spew.Dump(object_if)
		var ok bool
		var namespaces_if interface{}
		namespaces_if, ok = object_if["namespaces"]
		if ok {
			var namespaces_map map[string]interface{}
			namespaces_map, ok = namespaces_if.(map[string]interface{})
			if !ok {
				err = fmt.Errorf("(Parse_cwl_document) namespaces_if error (type: %s)", reflect.TypeOf(namespaces_if))
				return
			}
			namespaces = make(map[string]string)

			for key, value := range namespaces_map {

				switch value := value.(type) {
				case string:
					namespaces[key] = value
				default:
					err = fmt.Errorf("(Parse_cwl_document) value is not string (%s)", reflect.TypeOf(value))
					return
				}

			}

		} else {
			namespaces = nil
			//fmt.Println("no namespaces")
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
			err = fmt.Errorf("(Parse_cwl_document) GetId returns %s", err.Error())
			return
		}
		//fmt.Printf("this_id: %s\n", this_id)

		var object CWL_object
		var schemata_new []CWLType_Type
		object, schemata_new, err = New_CWL_object(object_if, cwl_version, nil, namespaces, nil)
		if err != nil {
			err = fmt.Errorf("(Parse_cwl_document) B New_CWL_object returns %s", err.Error())
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
			cwl_version = this_workflow.CwlVersion
		case *CommandLineTool:
			this_clt, _ := object.(*CommandLineTool)
			cwl_version = this_clt.CwlVersion
		case *ExpressionTool:
			this_et, _ := object.(*ExpressionTool)
			cwl_version = this_et.CwlVersion
		default:

			err = fmt.Errorf("(Parse_cwl_document) type unkown: %s", reflect.TypeOf(object))
			return
		}

		named_obj := NewNamed_CWL_object(this_id, object)
		//named_obj := NewNamed_CWL_object(commandlinetool.Id, commandlinetool)

		//cwl_version = commandlinetool.CwlVersion // TODO

		object_array = append(object_array, named_obj)
		for i, _ := range schemata_new {
			schemata = append(schemata, schemata_new[i])
		}

	}

	if namespaces == nil {
		panic("no namespaces")
	}

	//spew.Dump(object_array)
	//panic("done")
	return
}

func Add_to_collection_deprecated(collection *CWL_collection, object_array CWL_object_array) (err error) {

	for i, object := range object_array {
		err = collection.Add(strconv.Itoa(i), object) // TODO fix id
		if err != nil {
			err = fmt.Errorf("(Add_to_collection) collection.Add returned: %s", err.Error())
			return
		}
	}

	return
}

func Unmarshal(data_ptr *[]byte, v interface{}) (err error) {

	data := *data_ptr

	if data[0] == '{' {

		err_json := json.Unmarshal(data, v)
		if err_json != nil {
			logger.Debug(1, "CWL json unmarshal error: "+err_json.Error())
			err = err_json
			return
		}
	} else {
		err_yaml := yaml.Unmarshal(data, v)
		if err_yaml != nil {
			logger.Debug(1, "CWL yaml unmarshal error: "+err_yaml.Error())
			err = err_yaml
			return
		}

	}

	return
}
