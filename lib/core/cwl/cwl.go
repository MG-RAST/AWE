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

//type CWL_document struct {
//	CwlVersion string       `yaml:"cwlVersion"`
//	Graph      []CWL_object `yaml:"graph"`
//}

type CWL_object_generic map[string]interface{}

type CWL_object interface {
	Class() string
}

type Expression string

type CWLVersion interface{} // TODO

type Any interface{}

type LinkMergeMethod string // merge_nested or merge_flattened

type File struct {
	Path           string `yaml:"path"`
	Checksum       string `yaml:"checksum"`
	Size           int32  `yaml:"size"`
	SecondaryFiles []File `yaml:"secondaryFiles"`
	Format         string `yaml:"format"`
}

func Parse_cwl_document(yaml_str string) (err error, Workflows []Workflow, CommandLineTools []CommandLineTool) {

	// TODO check cwlVersion

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

		switch elem["hints"].(type) {
		case map[interface{}]interface{}:
			// Convert map of outputs into array of outputs
			err, elem["hints"] = CreateRequirementArray(elem["hints"])
			if err != nil {
				return
			}
		}

		switch {
		case cwl_object_type == "CommandLineTool":

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
			CommandLineTools = append(CommandLineTools, result)
			//container = append(container, result)
		case cwl_object_type == "Workflow":

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

			//spew.Dump(workflow)
			Workflows = append(Workflows, workflow)
			//container = append(container, result)
		} // end switch

		fmt.Printf("----------------------------------------------\n")
		//spew.Dump(CommandLineTools)
		//spew.Dump(Workflows)

		// pretty print json
		//b, err := json.MarshalIndent(CommandLineTools, "", "    ")
		//if err != nil {
		//		fmt.Println(err)
		//	return
		//}
		//fmt.Println(string(b))
		//t := elem.(CommandLineTool)

		//spew.Dump(t)

		//fmt.Println("A elem: " + elem.Class)
		//test_map := elem.(map[string]CWL_class)
		//test_map := elem.(map[string]interface{})
		//test_obj := test_map.(CWL_class)
		//fmt.Println("B test_map:")
		//spew.Dump(test_map)

		//value := test_map["class"]
		//fmt.Println("C")
		//value_str := value.(string)
		//fmt.Println("got: " + value_str)

	} // end for

	return
}

// func CreateAnyArray(original interface{}) (err error, new_array []Any) {
// 	//fmt.Printf("CreateAnyArray :::::::::::::::::::")
//
// 	for k, v := range original.(map[interface{}]interface{}) {
// 		//fmt.Printf("A")
//
// 		switch v.(type) {
// 		case map[interface{}]interface{}: // the hint is a struct itself
// 			fmt.Printf("match")
// 			vmap := v.(map[interface{}]interface{})
// 			vmap["id"] = k.(string)
// 			new_array = append(new_array, vmap)
// 		default:
// 			fmt.Printf("not match")
// 			return errors.New("error"), nil
// 		}
//
// 	}
// 	//spew.Dump(new_array)
// 	return
// }
