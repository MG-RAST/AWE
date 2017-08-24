package cwl

import (
	"encoding/json"
	"errors"
	"fmt"
	cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
	//"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v2"
	//"io/ioutil"
	//"os"
	_ "reflect"
	"strings"
)

// this is used by YAML or JSON library for inital parsing
type CWL_document_generic struct {
	CwlVersion CWLVersion           `yaml:"cwlVersion"`
	Graph      []CWL_object_generic `yaml:"graph"`
}

type CWL_object_generic map[string]interface{}

type CWLVersion string

type LinkMergeMethod string // merge_nested or merge_flattened

func New_CWL_object(elem map[string]interface{}, cwl_version CWLVersion) (obj cwl_types.CWL_object, err error) {

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

	switch cwl_object_type {
	case "CommandLineTool":
		logger.Debug(1, "parse CommandLineTool")
		result, xerr := NewCommandLineTool(elem)
		if xerr != nil {
			err = fmt.Errorf("NewCommandLineTool returned: %s", xerr)
			return
		}

		result.CwlVersion = cwl_version

		obj = result
		return
	case "Workflow":
		logger.Debug(1, "parse Workflow")
		workflow, xerr := NewWorkflow(elem)
		if xerr != nil {
			err = xerr
			return
		}
		obj = &workflow
		return
	} // end switch

	cwl_type, xerr := cwl_types.NewCWLType(cwl_object_id, elem)
	if xerr != nil {
		err = xerr
		return
	}
	obj = cwl_type

	return
}

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

	cwl_version := cwl_gen.CwlVersion

	// iterated over Graph
	for _, elem := range cwl_gen.Graph {
		object, xerr := New_CWL_object(elem, cwl_version)
		if xerr != nil {
			err = xerr
			return
		}

		collection.Add(object)

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
