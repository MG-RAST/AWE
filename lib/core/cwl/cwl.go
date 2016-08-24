package cwl

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
)

type CWL_document_generic struct {
	CwlVersion string               `yaml:"cwlVersion"`
	Graph      []CWL_object_generic `yaml:"graph"`
}

//type CWL_document struct {
//	CwlVersion string               `yaml:"cwlVersion"`
//	Graph      []CWL_object_generic `yaml:"graph"`
//}

type CWL_object_generic map[string]interface{}

type CWL_object interface {
	Class() string
}

type Workflow struct {
	Id string `yaml:"id"`
	//Class string `yaml:"class"`

	Inputs       []InputParameter          `yaml:"inputs"`
	Outputs      []WorkflowOutputParameter `yaml:"outputs"`
	Requirements interface{}               `yaml:"requirements"`
	Hints        interface{}               `yaml:"hints"`
	Label        string                    `yaml:"label"`
	Doc          string                    `yaml:"doc"`
	CwlVersion   CWLVersion                `yaml:"cwlVersion"`
}

func (w Workflow) Class() string { return "Workflow" }

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

type CommandLineTool struct {
	Id string `yaml:"id"`
	//Class string `yaml:"class"`

	BaseCommand        string                   `yaml:"baseCommand"` // TODO also allow []string
	Inputs             []CommandInputParameter  `yaml:"inputs"`
	Ouputs             []CommandOutputParameter `yaml:"outputs"`
	Hints              interface{}              `yaml:"hints"` // TODO Any
	Label              string                   `yaml:"label"`
	Description        string                   `yaml:"description"`
	CwlVersion         CWLVersion               `yaml:"cwlVersion"`
	Arguments          []string                 `yaml:"arguments"` // TODO support CommandLineBinding
	Stdin              string                   `yaml:"stdin"`     // TODO support Expression
	Stdout             string                   `yaml:"stdout"`    // TODO support Expression
	SuccessCodes       []int                    `yaml:"successCodes"`
	TemporaryFailCodes []int                    `yaml:"temporaryFailCodes"`
	PermanentFailCodes []int                    `yaml:"permanentFailCodes"`
}

func (c CommandLineTool) Class() string { return "CommandLineTool" }

type DockerRequirement struct {
	//Class         string `yaml:"class"`
	DockerPull    string `yaml:"dockerPull"`
	DockerLoad    string `yaml:"dockerLoad"`
	DockerFile    string `yaml:"dockerFile"`
	DockerImport  string `yaml:"dockerImport"`
	DockerImageId string `yaml:"dockerImageId"`
}

type Expression string
type Any interface{}        // TODO
type CWLVersion interface{} // TODO

type LinkMergeMethod interface{} // TODO

type File struct {
	Path           string `yaml:"path"`
	Checksum       string `yaml:"checksum"`
	Size           int32  `yaml:"size"`
	SecondaryFiles []File `yaml:"secondaryFiles"`
	Format         string `yaml:"format"`
}

type CommandInputParameter struct {
	Id             string   `yaml:"id"`
	SecondaryFiles []string `yaml:"secondaryFiles"` // TODO string | Expression | array<string | Expression>
	Format         string   `yaml:"format"`
	Streamable     bool     `yaml:"streamable"`
	Type           string   `yaml:"type"` // TODO CWLType | CommandInputRecordSchema | CommandInputEnumSchema | CommandInputArraySchema | string | array<CWLType | CommandInputRecordSchema | CommandInputEnumSchema | CommandInputArraySchema | string>
	Label          string   `yaml:"label"`
	Description    string   `yaml:"description"`
	InputBinding   string   `yaml:"inputBinding"` //TODO
	Default        Any      `yaml:"default"`
}

type CommandOutputParameter struct {
	Id             string   `yaml:"id"`
	SecondaryFiles []string `yaml:"secondaryFiles"` // TODO string | Expression | array<string | Expression>
	Format         string   `yaml:"format"`
	Streamable     bool     `yaml:"streamable"`
	Type           string   `yaml:"type"` // TODO CWLType | CommandInputRecordSchema | CommandInputEnumSchema | CommandInputArraySchema | string | array<CWLType | CommandInputRecordSchema | CommandInputEnumSchema | CommandInputArraySchema | string>
	Label          string   `yaml:"label"`
	Description    string   `yaml:"description"`
	OutputBinding  string   `yaml:"inputBinding"` //TODO
}

// keyname will be converted into 'Id'-field
func CreateCommandInputArray(original interface{}) []CommandInputParameter {

	var new_array []CommandInputParameter

	for k, v := range original.(map[interface{}]interface{}) {
		fmt.Printf("A")

		var input_parameter CommandInputParameter
		mapstructure.Decode(v, &input_parameter)
		input_parameter.Id = k.(string)
		fmt.Printf("C")
		new_array = append(new_array, input_parameter)
		fmt.Printf("D")

	}
	spew.Dump(new_array)
	return new_array
}

// keyname will be converted into 'Id'-field
func CreateCommandOutputArray(original interface{}) []CommandOutputParameter {
	fmt.Printf("CreateOutputMap")
	var new_array []CommandOutputParameter

	for k, v := range original.(map[interface{}]interface{}) {
		fmt.Printf("A")

		var output_parameter CommandOutputParameter
		mapstructure.Decode(v, &output_parameter)
		output_parameter.Id = k.(string)
		fmt.Printf("C")
		new_array = append(new_array, output_parameter)
		fmt.Printf("D")

	}
	spew.Dump(new_array)
	return new_array
}

// InputParameter
func CreateInputParameterArray(original interface{}) []InputParameter {

	var new_array []InputParameter

	for k, v := range original.(map[interface{}]interface{}) {
		fmt.Printf("A")

		var input_parameter InputParameter
		mapstructure.Decode(v, &input_parameter)
		input_parameter.Id = k.(string)
		fmt.Printf("C")
		new_array = append(new_array, input_parameter)
		fmt.Printf("D")

	}
	spew.Dump(new_array)
	return new_array
}

// WorkflowOutputParameter
func CreateWorkflowOutputParameterArray(original interface{}) []WorkflowOutputParameter {

	var new_array []WorkflowOutputParameter

	for k, v := range original.(map[interface{}]interface{}) {
		fmt.Printf("A")

		var output_parameter WorkflowOutputParameter
		mapstructure.Decode(v, &output_parameter)
		output_parameter.Id = k.(string)
		fmt.Printf("C")
		new_array = append(new_array, output_parameter)
		fmt.Printf("D")

	}
	spew.Dump(new_array)
	return new_array
}
