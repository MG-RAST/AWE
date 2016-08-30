package cwl

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v2"
	//"os"
	//"reflect"
	"strings"
)

// this is used by YAML or JSON library for inital parsing
type CWL_document_generic struct {
	CwlVersion string               `yaml:"cwlVersion"`
	Graph      []CWL_object_generic `yaml:"graph"`
}

type CWL_object interface {
	GetClass() string
	GetId() string
}

type CWL_object_generic map[string]interface{}

type Expression string

type CWLVersion interface{} // TODO

type Any interface{}

type LinkMergeMethod string // merge_nested or merge_flattened

type File struct {
	Id             string `yaml:"id"`
	Path           string `yaml:"path"`
	Checksum       string `yaml:"checksum"`
	Size           int32  `yaml:"size"`
	SecondaryFiles []File `yaml:"secondaryFiles"`
	Format         string `yaml:"format"`
}

func (f File) GetClass() string { return "File" }
func (f File) GetId() string    { return f.Id }

func Parse_cwl_document(yaml_str string) (err error, collection CWL_collection) {

	// TODO check cwlVersion
	// TODO screen for "$import": // this might break the YAML parser !

	collection = NewCWL_collection()

	// this yaml parser (gopkg.in/yaml.v2) has problems with the CWL yaml format. We skip the header aand jump directly to "$graph" because of that.
	graph_pos := strings.Index(yaml_str, "$graph:")

	if graph_pos == -1 {
		err = errors.New("yaml parisng error. keyword $graph missing")
		return
	}

	yaml_str = strings.Replace(yaml_str, "$graph", "graph", -1) // remove dollar sign

	cwl_gen := CWL_document_generic{}

	err = yaml.Unmarshal([]byte(yaml_str), &cwl_gen)
	if err != nil {
		logger.Debug(1, "CWL unmarshal error")
		logger.Error("error: " + err.Error())
	}

	fmt.Println("-------------- raw CWL")
	spew.Dump(cwl_gen)
	fmt.Println("-------------- Start real parsing")

	// iterated over Graph
	for _, elem := range cwl_gen.Graph {

		cwl_object_type := elem["class"].(string)

		cwl_object_id := elem["id"].(string)
		_ = cwl_object_id
		switch elem["hints"].(type) {
		case map[interface{}]interface{}:
			// Convert map of outputs into array of outputs
			err, elem["hints"] = CreateRequirementArray(elem["hints"])
			if err != nil {
				return
			}
		}

		switch cwl_object_type {
		case "CommandLineTool":

			//*** check if "inputs"" is an array or a map"
			switch elem["inputs"].(type) {
			case map[interface{}]interface{}:
				// Convert map of inputs into array of inputs
				err, elem["inputs"] = CreateCommandInputArray(elem["inputs"])
				if err != nil {
					return
				}
			}

			switch elem["outputs"].(type) {
			case map[interface{}]interface{}:
				// Convert map of outputs into array of outputs
				err, elem["outputs"] = CreateCommandOutputArray(elem["outputs"])
				if err != nil {
					return
				}
			}

			var result CommandLineTool
			err = mapstructure.Decode(elem, &result)
			if err != nil {
				return
			}
			spew.Dump(result)
			collection.CommandLineTools[result.Id] = result
			//collection = append(collection, result)
		case "Workflow":

			// convert input map into input array
			switch elem["inputs"].(type) {
			case map[interface{}]interface{}:
				// Convert map of inputs into array of inputs
				err, elem["inputs"] = CreateInputParameterArray(elem["inputs"])
				if err != nil {
					return
				}
			}

			switch elem["outputs"].(type) {
			case map[interface{}]interface{}:
				// Convert map of outputs into array of outputs
				err, elem["outputs"] = CreateWorkflowOutputParameterArray(elem["outputs"])
				if err != nil {
					return
				}
			}

			// convert steps to array if it is a map
			switch elem["steps"].(type) {
			case map[interface{}]interface{}:
				err, elem["steps"] = CreateWorkflowStepsArray(elem["steps"])
				if err != nil {
					return
				}
			}

			switch elem["requirements"].(type) {
			case map[interface{}]interface{}:
				// Convert map of outputs into array of outputs
				err, elem["requirements"] = CreateRequirementArray(elem["requirements"])
				if err != nil {
					return
				}
			}
			//fmt.Printf("-- Steps found ------------") // WorkflowStep
			//for _, step := range elem["steps"].([]interface{}) {

			//	spew.Dump(step)

			//}

			var workflow Workflow
			err = mapstructure.Decode(elem, &workflow)
			if err != nil {
				return
			}

			for _, input := range workflow.Inputs {
				// input is InputParameter
				collection.add(input)
			}

			//spew.Dump(workflow)
			collection.add(workflow)
			//collection.Workflows = append(collection.Workflows, workflow)
			//collection = append(collection, result)
		case "File":
			var cwl_file File
			err = mapstructure.Decode(elem, &cwl_file)
			if err != nil {
				return
			}
			if cwl_file.Id == "" {
				cwl_file.Id = cwl_object_id
			}
			//collection.Files[cwl_file.Id] = cwl_file
			collection.add(cwl_file)
		default:
			err = errors.New("object unknown")
			return
		} // end switch

		fmt.Printf("----------------------------------------------\n")

	} // end for

	return
}
