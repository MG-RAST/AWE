package cwl

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v2"
	//"io/ioutil"
	//"os"
	_ "reflect"
	"strings"
)

// http://www.commonwl.org/draft-3/CommandLineTool.html#CWLType
const (
	CWL_null    = "null"    //no value
	CWL_boolean = "boolean" //a binary value
	CWL_int     = "int"     //32-bit signed integer
	CWL_long    = "long"    //64-bit signed integer
	CWL_float   = "float"   //single precision (32-bit) IEEE 754 floating-point number
	CWL_double  = "double"  //double precision (64-bit) IEEE 754 floating-point number
	CWL_string  = "string"  //Unicode character sequence
	CWL_File    = "File"    //A File object

	CWL_stdout = "stdout"
	CWL_stderr = "stderr"

	CWL_null_array    = "null_array"    //no value
	CWL_boolean_array = "boolean_array" //a binary value
	CWL_int_array     = "int_array"     //32-bit signed integer
	CWL_long_array    = "long_array"    //64-bit signed integer
	CWL_float_array   = "float_array"   //single precision (32-bit) IEEE 754 floating-point number
	CWL_double_array  = "double_array"  //double precision (64-bit) IEEE 754 floating-point number
	CWL_string_array  = "string_array"  //Unicode character sequence
	CWL_File_array    = "File_array"    //A File object

)

var valid_cwltypes = map[string]bool{
	CWL_null:    true,
	CWL_boolean: true,
	CWL_int:     true,
	CWL_long:    true,
	CWL_float:   true,
	CWL_double:  true,
	CWL_string:  true,
	CWL_File:    true,
	CWL_stdout:  true,
	CWL_stderr:  true,
}

type CWL_minimal_interface interface {
	is_CWL_minimal()
}

type CWL_minimal struct{}

func (c *CWL_minimal) is_CWL_minimal() {}

// this is used by YAML or JSON library for inital parsing
type CWL_document_generic struct {
	CwlVersion string               `yaml:"cwlVersion"`
	Graph      []CWL_object_generic `yaml:"graph"`
}

type CWL_object_generic map[string]interface{}

type CWLVersion interface{} // TODO

// generic class to represent Files and Directories
type CWL_location interface {
	GetLocation() string
}

type LinkMergeMethod string // merge_nested or merge_flattened

func Parse_cwl_document(collection *CWL_collection, yaml_str string) (err error) {

	// TODO check cwlVersion
	// TODO screen for "$import": // this might break the YAML parser !

	// this yaml parser (gopkg.in/yaml.v2) has problems with the CWL yaml format. We skip the header aand jump directly to "$graph" because of that.
	graph_pos := strings.Index(yaml_str, "$graph")

	if graph_pos == -1 {
		err = errors.New("yaml parisng error. keyword $graph missing: ")
		return
	}

	yaml_str = strings.Replace(yaml_str, "$graph", "graph", -1) // remove dollar sign

	cwl_gen := CWL_document_generic{}

	err = Unmarshal([]byte(yaml_str), &cwl_gen)
	if err != nil {
		logger.Debug(1, "CWL unmarshal error")
		logger.Error("error: " + err.Error())
	}

	fmt.Println("-------------- raw CWL")
	spew.Dump(cwl_gen)
	fmt.Println("-------------- Start real parsing")

	// iterated over Graph
	for _, elem := range cwl_gen.Graph {

		cwl_object_type, ok := elem["class"].(string)

		if !ok {
			err = errors.New("object has no member class")
			return
		}

		cwl_object_id := elem["id"].(string)
		if !ok {
			err = errors.New("object has no member id")
			return
		}
		_ = cwl_object_id
		//switch elem["hints"].(type) {
		//case map[interface{}]interface{}:
		//logger.Debug(1, "hints is map[interface{}]interface{}")
		// Convert map of outputs into array of outputs
		//err, elem["hints"] = CreateRequirementArray(elem["hints"])
		//if err != nil {
		//	return
		//}
		//case interface{}[]:
		//	logger.Debug(1, "hints is interface{}[]")
		//default:
		//	logger.Debug(1, "hints is something else")
		//}

		switch cwl_object_type {
		case "CommandLineTool":
			logger.Debug(1, "parse CommandLineTool")
			result, xerr := NewCommandLineTool(elem)
			if xerr != nil {
				err = fmt.Errorf("NewCommandLineTool returned: %s", xerr)
				return
			}
			//*** check if "inputs"" is an array or a map"

			//collection.CommandLineTools[result.Id] = result
			err = collection.Add(result)
			if err != nil {
				return
			}
			//collection = append(collection, result)
		case "Workflow":
			logger.Debug(1, "parse Workflow")
			workflow, xerr := NewWorkflow(elem, collection)
			if xerr != nil {
				err = xerr
				return
			}

			// some checks and add inputs to collection
			for _, input := range workflow.Inputs {
				// input is InputParameter

				if input.Id == "" {
					err = fmt.Errorf("input has no ID")
					return
				}

				//if !strings.HasPrefix(input.Id, "#") {
				//	if !strings.HasPrefix(input.Id, "inputs.") {
				//		input.Id = "inputs." + input.Id
				//	}
				//}
				err = collection.Add(input)
				if err != nil {
					return
				}
			}

			//fmt.Println("WORKFLOW")
			//spew.Dump(workflow)
			err = collection.Add(&workflow)
			if err != nil {
				return
			}

			//collection.Workflows = append(collection.Workflows, workflow)
			//collection = append(collection, result)
		case "File":
			logger.Debug(1, "parse File")
			var cwl_file File
			err = mapstructure.Decode(elem, &cwl_file)
			if err != nil {
				err = fmt.Errorf("(Parse_cwl_document/File) %s", err.Error())
				return
			}
			if cwl_file.Id == "" {
				cwl_file.Id = cwl_object_id
			}
			//collection.Files[cwl_file.Id] = cwl_file
			err = collection.Add(&cwl_file)
			if err != nil {
				return
			}
		default:
			err = errors.New("object unknown")
			return
		} // end switch

		fmt.Printf("----------------------------------------------\n")

	} // end for

	return
}

func Unmarshal(data []byte, v interface{}) (err error) {
	err_yaml := yaml.Unmarshal(data, v)
	if err_yaml != nil {
		logger.Debug(1, "CWL YAML unmarshal error, (try json...) : "+err_yaml.Error())
		err_json := json.Unmarshal(data, v)
		if err_json != nil {
			logger.Debug(1, "CWL JSON unmarshal error: "+err_json.Error())
		}
	}

	if err != nil {
		err = errors.New("Could not parse document as JSON or YAML")
	}

	return
}
