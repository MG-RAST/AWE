package cwl

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
	//"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v2"
	//"io/ioutil"
	//"os"
	"reflect"
	"strconv"
	"strings"
)

// this is used by YAML or JSON library for inital parsing
type CWL_document_generic struct {
	CwlVersion CWLVersion    `yaml:"cwlVersion"`
	Graph      []interface{} `yaml:"graph"`
	//Graph      []CWL_object_generic `yaml:"graph"`
}

type CWL_object_generic map[string]interface{}

type CWLVersion string

type LinkMergeMethod string // merge_nested or merge_flattened

func New_CWL_object(original interface{}, cwl_version CWLVersion) (obj CWL_object, schemata []CWLType_Type, err error) {
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
		cwl_object_type, ok := elem["class"].(string)

		if !ok {
			fmt.Println("------------------")
			spew.Dump(original)
			err = errors.New("(New_CWL_object) object has no field \"class\"")
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
			clt, schemata, err = NewCommandLineTool(elem)
			if err != nil {
				err = fmt.Errorf("(New_CWL_object) NewCommandLineTool returned: %s", err.Error())
				return
			}

			clt.CwlVersion = cwl_version
			obj = clt

			return
		case "Workflow":
			//fmt.Println("New_CWL_object Workflow")
			logger.Debug(1, "parse Workflow")
			obj, schemata, err = NewWorkflow(elem)
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

func NewCWL_object_array(original interface{}) (array CWL_object_array, schemata []CWLType_Type, err error) {

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
			cwl_object, schemata_new, err = New_CWL_object(element, "")
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

func Parse_cwl_document(yaml_str string) (object_array Named_CWL_object_array, cwl_version CWLVersion, schemata []CWLType_Type, err error) {

	graph_pos := strings.Index(yaml_str, "$graph")

	if graph_pos != -1 {

		yaml_str = strings.Replace(yaml_str, "$graph", "graph", -1) // remove dollar sign

		cwl_gen := CWL_document_generic{}

		yaml_byte := []byte(yaml_str)
		err = Unmarshal(&yaml_byte, &cwl_gen)
		if err != nil {
			logger.Debug(1, "CWL unmarshal error")
			logger.Error("error: " + err.Error())
		}

		//fmt.Println("-------------- raw CWL")
		//spew.Dump(cwl_gen)
		//fmt.Println("-------------- Start parsing")

		cwl_version = cwl_gen.CwlVersion

		// iterated over Graph

		//fmt.Println("-------------- A Parse_cwl_document")
		for count, elem := range cwl_gen.Graph {
			//fmt.Println("-------------- B Parse_cwl_document")

			var id string
			id, err = GetId(elem)
			if err != nil {
				return
			}

			var object CWL_object
			var schemata_new []CWLType_Type
			object, schemata_new, err = New_CWL_object(elem, cwl_version)
			if err != nil {
				err = fmt.Errorf("(Parse_cwl_document) New_CWL_object returns %s", err.Error())
				return
			}

			named_obj := NewNamed_CWL_object(id, object)
			//fmt.Println("-------------- C Parse_cwl_document")
			object_array = append(object_array, named_obj)

			for i, _ := range schemata_new {
				schemata = append(schemata, schemata_new[i])
			}

			logger.Debug(3, "Added %d cwl objects...", count)
			//fmt.Println("-------------- loop Parse_cwl_document")
		} // end for

		//fmt.Println("-------------- finished Parse_cwl_document")

	} else {

		var commandlinetool_if map[string]interface{}
		yaml_byte := []byte(yaml_str)
		err = Unmarshal(&yaml_byte, &commandlinetool_if)
		if err != nil {
			logger.Debug(1, "CWL unmarshal error")
			logger.Error("error: " + err.Error())
		}

		//fmt.Println("-------------- raw CWL")
		//spew.Dump(commandlinetool_if)
		//fmt.Println("-------------- Start parsing")

		var commandlinetool *CommandLineTool
		var schemata_new []CWLType_Type
		commandlinetool, schemata_new, err = NewCommandLineTool(commandlinetool_if)
		if err != nil {
			err = fmt.Errorf("(Parse_cwl_document) NewCommandLineTool returned: %s", err.Error())
			return
		}

		named_obj := NewNamed_CWL_object(commandlinetool.Id, commandlinetool)

		cwl_version = commandlinetool.CwlVersion
		object_array = append(object_array, named_obj)
		for i, _ := range schemata_new {
			schemata = append(schemata, schemata_new[i])
		}

	}

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
