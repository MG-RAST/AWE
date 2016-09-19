package cwl

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
	//"os"
	"reflect"
	"strings"
)

type Workflow struct {
	Inputs       []InputParameter          `yaml:"inputs"`
	Outputs      []WorkflowOutputParameter `yaml:"outputs"`
	Id           string                    `yaml:"id"`
	Steps        []WorkflowStep            `yaml:"steps"`
	Requirements []Requirement             `yaml:"requirements"`
	Hints        []Requirement             `yaml:"hints"` // TODO Hints may contain non-requirement objects. Give warning in those cases.
	Label        string                    `yaml:"label"`
	Doc          string                    `yaml:"doc"`
	CwlVersion   CWLVersion                `yaml:"cwlVersion"`
	Metadata     map[string]interface{}    `yaml:"metadata"`
}

func (w Workflow) GetClass() string { return "Workflow" }
func (w Workflow) GetId() string    { return w.Id }

type WorkflowStep struct {
	Id            string               `yaml:"id"`
	In            []WorkflowStepInput  `yaml:"in"` // array<WorkflowStepInput> | map<WorkflowStepInput.id, WorkflowStepInput.source> | map<WorkflowStepInput.id, WorkflowStepInput>
	Out           []WorkflowStepOutput `yaml:"out"`
	Run           string               `yaml:"run"` // Specification unclear: string | CommandLineTool | ExpressionTool | Workflow
	Requirements  []Requirement        `yaml:"requirements"`
	Hints         []Requirement        `yaml:"hints"`
	Label         string               `yaml:"label"`
	Doc           string               `yaml:"doc"`
	Scatter       string               `yaml:"scatter"`       // ScatterFeatureRequirement
	ScatterMethod string               `yaml:"scatterMethod"` // ScatterFeatureRequirement
}

//http://www.commonwl.org/v1.0/Workflow.html#WorkflowStepInput
type WorkflowStepInput struct {
	Id        string          `yaml:"id"`
	Source    []string        `yaml:"source"` // MultipleInputFeatureRequirement
	LinkMerge LinkMergeMethod `yaml:"linkMerge"`
	Default   Any             `yaml:"default"`   // type Any does not make sense
	ValueFrom Expression      `yaml:"valueFrom"` // this is a StepInputExpressionRequirement
}

type WorkflowStepOutput struct {
	Id string `yaml:"id"`
}

type InputParameter struct {
	Id             string             `yaml:"id"`
	Label          string             `yaml:"label"`
	SecondaryFiles []string           `yaml:"secondaryFiles"` // TODO string | Expression | array<string | Expression>
	Format         string             `yaml:"format"`
	Streamable     bool               `yaml:"streamable"`
	Doc            string             `yaml:"doc"`
	InputBinding   CommandLineBinding `yaml:"inputBinding"` //TODO
	Default        Any                `yaml:"default"`
	Type           string             `yaml:"type"` // TODO CWLType | InputRecordSchema | InputEnumSchema | InputArraySchema | string | array<CWLType | InputRecordSchema | InputEnumSchema | InputArraySchema | string>
}

func (i InputParameter) GetClass() string { return "InputParameter" }
func (i InputParameter) GetId() string    { return i.Id }

type WorkflowOutputParameter struct {
	Id             string               `yaml:"id"`
	Label          string               `yaml:"label"`
	SecondaryFiles []Expression         `yaml:"secondaryFiles"` // TODO string | Expression | array<string | Expression>
	Format         []Expression         `yaml:"format"`
	Streamable     bool                 `yaml:"streamable"`
	Doc            string               `yaml:"doc"`
	OutputBinding  CommandOutputBinding `yaml:"outputBinding"` //TODO
	OutputSource   []string             `yaml:"outputSource"`
	LinkMerge      LinkMergeMethod      `yaml:"linkMerge"`
	Type           string               `yaml:"type"` // TODO CWLType | OutputRecordSchema | OutputEnumSchema | OutputArraySchema | string | array<CWLType | OutputRecordSchema | OutputEnumSchema | OutputArraySchema | string>
}

//http://www.commonwl.org/v1.0/Workflow.html#CommandLineBinding
type CommandLineBinding struct {
	LoadContents  bool   `yaml:"loadContents"`
	Position      int    `yaml:"position"`
	Prefix        string `yaml:"prefix"`
	Separate      string `yaml:"separate"`
	ItemSeparator string `yaml:"itemSeparator"`
	ValueFrom     string `yaml:"valueFrom"`
	ShellQuote    bool   `yaml:"shellQuote"`
}

type CommandOutputBinding struct {
	Glob         []Expression `yaml:"glob"`
	LoadContents bool         `yaml:"loadContents"`
	OutputEval   Expression   `yaml:"outputEval"`
}

// InputParameter
func CreateInputParameterArray(original interface{}) (err error, new_array []InputParameter) {

	for k, v := range original.(map[interface{}]interface{}) {
		//fmt.Printf("A")

		var input_parameter InputParameter
		id, ok := k.(string)
		if !ok {
			err = fmt.Errorf("Cannot parse id of input")
			return
		}
		switch v.(type) {
		case string:
			type_string := v.(string)
			switch strings.ToLower(type_string) {
			case "string":
				input_parameter = InputParameter{Id: id, Type: "string"}
			case "file":
				input_parameter = InputParameter{Id: id, Type: "file"}
			default:
				err = fmt.Errorf("unknown type: \"%s\"", type_string)
				return
			}
		case int:
			input_parameter = InputParameter{Id: id, Type: "int"}
		case map[interface{}]interface{}:
			mapstructure.Decode(v, &input_parameter)
		default:
			err = fmt.Errorf("cannot parse input \"%s\"", id)
			return
		}

		if input_parameter.Id == "" {
			input_parameter.Id = id
		}

		if input_parameter.Id == "" {
			err = fmt.Errorf("ID is missing", id)
			return
		}

		if input_parameter.Type == "" {
			spew.Dump(v)
			err = fmt.Errorf("Type not known \"%s\"", id)
			return
		}

		//fmt.Printf("C")
		new_array = append(new_array, input_parameter)
		//fmt.Printf("D")

	}
	//spew.Dump(new_array)
	//os.Exit(0)
	return
}

// WorkflowOutputParameter
func CreateWorkflowOutputParameterArray(original interface{}) (err error, new_array []WorkflowOutputParameter) {

	for k, v := range original.(map[interface{}]interface{}) {
		//fmt.Printf("A")

		var output_parameter WorkflowOutputParameter
		mapstructure.Decode(v, &output_parameter)
		output_parameter.Id = k.(string)
		//fmt.Printf("C")
		new_array = append(new_array, output_parameter)
		//fmt.Printf("D")

	}
	//spew.Dump(new_array)
	return
}

// CreateWorkflowStepsArray
func CreateWorkflowStepsArray(original interface{}) (err error, new_array []WorkflowStep) {

	// iterate over workflow steps
	for k, v := range original.(map[interface{}]interface{}) {
		fmt.Printf("A step\n")
		spew.Dump(v)

		fmt.Println("type: ")
		fmt.Println(reflect.TypeOf(v))

		v_map := v.(map[interface{}]interface{})
		spew.Dump(v_map)

		switch v_map["in"].(type) {
		case map[interface{}]interface{}:
			err, v_map["in"] = CreateWorkflowStepInputArray(v_map["in"])
			if err != nil {
				return
			}
		}

		switch v_map["out"].(type) {
		case map[interface{}]interface{}: // example {'x'->?} -> [{id:'x'}]
			fmt.Printf("match map[interface{}]interface{}\n")
			err, v_map["out"] = CreateWorkflowStepOutputArray(v_map["out"])
			if err != nil {
				return
			}
		case []interface{}: // example ['x'] -> [{id:'x'}]
			fmt.Printf("match []string\n")
			err, v_map["out"] = CreateWorkflowStepOutputArray(v_map["out"])
			if err != nil {
				return
			}
		default:
			// TODO some error
			fmt.Printf("match default\n")
			fmt.Println("type: ")
			fmt.Println(reflect.TypeOf(v_map["out"]))
		}

		switch v_map["hints"].(type) {
		case map[interface{}]interface{}:
			// Convert map of outputs into array of outputs
			err, v_map["hints"] = CreateRequirementArray(v_map["hints"])
			if err != nil {
				return
			}
		}

		switch v_map["requirements"].(type) {
		case map[interface{}]interface{}:
			// Convert map of outputs into array of outputs
			err, v_map["requirements"] = CreateRequirementArray(v_map["requirements"])
			if err != nil {
				return
			}
		}

		var step WorkflowStep
		mapstructure.Decode(v, &step)
		step.Id = k.(string)

		fmt.Printf("Last step\n")
		spew.Dump(step)
		//fmt.Printf("C")
		new_array = append(new_array, step)
		//fmt.Printf("D")

	}
	//spew.Dump(new_array)
	return
}

func GetMapElement(m map[interface{}]interface{}, key string) (err error, value interface{}) {

	for k, v := range m {
		k_str, ok := k.(string)
		if ok {
			if k_str == key {
				value = v
				return
			}
		}
	}
	err = fmt.Errorf("Element in map not found")
	return
}

func CreateWorkflowStepInputArray(original interface{}) (err error, new_array []WorkflowStepInput) {

	input_array, ok := original.(map[interface{}]interface{})
	if !ok {
		err = fmt.Errorf("could not parse WorkflowStepInputArray")
		fmt.Println("original: ")
		spew.Dump(original)
		return
	}

	for k, v := range input_array {
		//v is an input

		var input_parameter WorkflowStepInput

		fmt.Println("::::::::::::::::::::")
		spew.Dump(v)
		spew.Dump(input_parameter)

		switch v.(type) {
		case string:
			fmt.Println("case string")
			source_string := v.(string)
			fmt.Printf("source_string: %s\n", source_string)
			input_parameter.Source = []string{"#" + source_string} // TODO this is a  WorkflowStepInput.source or WorkflowStepInput (the latter would not make much sense)
		case int:
			fmt.Println("int")
			input_parameter.Default = Int{Id: input_parameter.Id, Value: v.(int)}
		case map[interface{}]interface{}:
			fmt.Println("case map[interface{}]interface{}")
			mapstructure.Decode(v, &input_parameter)

			// take care of Default key:

			errx, default_value := GetMapElement(v.(map[interface{}]interface{}), "default")

			//fmt.Println("test:", test.(string))

			if errx != nil {
				return errx, new_array
			}

			switch default_value.(type) {
			case string:
				input_parameter.Default = String{Id: input_parameter.Id, Value: default_value.(string)}
			case int:
				input_parameter.Default = Int{Id: input_parameter.Id, Value: default_value.(int)}
			default:
				err = fmt.Errorf("string or int expected for key \"default\"")
				return
			}

		default:
			fmt.Println("case default")
			err = fmt.Errorf("Input type for %s can not be parsed", input_parameter.Id)
			return
		}

		if input_parameter.Id == "" {
			input_parameter.Id = k.(string)
		}
		fmt.Println("input_parameter:")
		spew.Dump(input_parameter)

		// now v should be a map

		fmt.Println("done.")
		new_array = append(new_array, input_parameter)
		//fmt.Printf("D")

	}
	//spew.Dump(new_array)
	return
}

func CreateWorkflowStepOutputArray(original interface{}) (err error, new_array []WorkflowStepOutput) {

	switch original.(type) {
	case map[interface{}]interface{}:

		for k, v := range original.(map[interface{}]interface{}) {
			//fmt.Printf("A")

			var output_parameter WorkflowStepOutput
			mapstructure.Decode(v, &output_parameter)

			output_parameter.Id = k.(string)
			//fmt.Printf("C")
			new_array = append(new_array, output_parameter)
			//fmt.Printf("D")

		}

	case []interface{}:
		for _, v := range original.([]interface{}) {

			switch v.(type) {
			case string:
				output_parameter := WorkflowStepOutput{Id: v.(string)}
				new_array = append(new_array, output_parameter)
			default:
				wso, ok := v.(WorkflowStepOutput)
				if !ok {
					// TODO some ERROR
				}
				new_array = append(new_array, wso)
			}

		}

	} // end switch

	//spew.Dump(new_array)
	return
}
