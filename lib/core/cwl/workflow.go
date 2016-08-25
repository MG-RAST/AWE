package cwl

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
	"reflect"
	"strings"
)

type Workflow struct {
	Inputs       []InputParameter          `yaml:"inputs"`
	Outputs      []WorkflowOutputParameter `yaml:"outputs"`
	Id           string                    `yaml:"id"`
	Steps        []WorkflowStep            `yaml:"steps"`
	Requirements []Requirement             `yaml:"requirements"`
	Hints        []Any                     `yaml:"hints"`
	Label        string                    `yaml:"label"`
	Doc          string                    `yaml:"doc"`
	CwlVersion   CWLVersion                `yaml:"cwlVersion"`
}

func (w Workflow) Class() string { return "Workflow" }

type WorkflowStep struct {
	Id            string               `yaml:"id"`
	In            []WorkflowStepInput  `yaml:"in"` // array<WorkflowStepInput> | map<WorkflowStepInput.id, WorkflowStepInput.source> | map<WorkflowStepInput.id, WorkflowStepInput>
	Out           []WorkflowStepOutput `yaml:"out"`
	Run           string               `yaml:"run"` // Specification unclear: string | CommandLineTool | ExpressionTool | Workflow
	Requirements  []Requirement        `yaml:"requirements"`
	Hints         []Any                `yaml:"hints"`
	Label         string               `yaml:"label"`
	Doc           string               `yaml:"doc"`
	Scatter       string               `yaml:"scatter"`
	ScatterMethod string               `yaml:"scatterMethod"`
}

type WorkflowStepInput struct {
	Id        string          `yaml:"id"`
	Source    []string        `yaml:"source"`
	LinkMerge LinkMergeMethod `yaml:"linkMerge"`
	Default   Any             `yaml:"default"`
	ValueFrom Expression      `yaml:"valueFrom"`
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
		mapstructure.Decode(v, &input_parameter)
		input_parameter.Id = k.(string)
		//fmt.Printf("C")
		new_array = append(new_array, input_parameter)
		//fmt.Printf("D")

	}
	//spew.Dump(new_array)
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
			err, v_map["hints"] = CreateAnyArray(v_map["hints"])
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

func CreateWorkflowStepInputArray(original interface{}) (err error, new_array []WorkflowStepInput) {

	for k, v := range original.(map[interface{}]interface{}) {
		//fmt.Printf("A")

		var input_parameter WorkflowStepInput
		mapstructure.Decode(v, &input_parameter)
		input_parameter.Id = k.(string)

		switch v.(type) {
		case string:
			source_string := v.(string)
			fmt.Printf("source_string: %s\n", source_string)
			if strings.HasPrefix(source_string, "#") {
				input_parameter.Source = []string{source_string} // TODO this is a  WorkflowStepInput.source or WorkflowStepInput (the latter would not make much sense)
			} else {
				input_parameter.Default = source_string //  TODO this should be an Expression, not just a string !?
			}
		default:
			input_parameter.Default = v
		}

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
