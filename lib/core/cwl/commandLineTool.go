package cwl

import (
	//"errors"
	//"fmt"
	"github.com/mitchellh/mapstructure"
)

type CommandLineTool struct {
	Id string `yaml:"id"`

	BaseCommand        string                   `yaml:"baseCommand"` // TODO also allow []string
	Inputs             []CommandInputParameter  `yaml:"inputs"`
	Ouputs             []CommandOutputParameter `yaml:"outputs"`
	Hints              []Requirement            `yaml:"hints"` // TODO Any
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

func (c CommandLineTool) GetClass() string { return "CommandLineTool" }
func (c CommandLineTool) GetId() string    { return c.Id }

type CommandInputParameter struct {
	Id             string             `yaml:"id"`
	SecondaryFiles []string           `yaml:"secondaryFiles"` // TODO string | Expression | array<string | Expression>
	Format         string             `yaml:"format"`
	Streamable     bool               `yaml:"streamable"`
	Type           string             `yaml:"type"` // TODO CWLType | CommandInputRecordSchema | CommandInputEnumSchema | CommandInputArraySchema | string | array<CWLType | CommandInputRecordSchema | CommandInputEnumSchema | CommandInputArraySchema | string>
	Label          string             `yaml:"label"`
	Description    string             `yaml:"description"`
	InputBinding   CommandLineBinding `yaml:"inputBinding"`
	Default        Any                `yaml:"default"`
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
func CreateCommandInputArray(original interface{}) (err error, new_array []CommandInputParameter) {

	for k, v := range original.(map[interface{}]interface{}) {
		//fmt.Printf("A")

		var input_parameter CommandInputParameter
		mapstructure.Decode(v, &input_parameter)
		input_parameter.Id = k.(string)
		//fmt.Printf("C")
		new_array = append(new_array, input_parameter)
		//fmt.Printf("D")

	}
	//spew.Dump(new_array)
	return
}

// keyname will be converted into 'Id'-field
func CreateCommandOutputArray(original interface{}) (err error, new_array []CommandOutputParameter) {

	for k, v := range original.(map[interface{}]interface{}) {
		//fmt.Printf("A")

		var output_parameter CommandOutputParameter
		mapstructure.Decode(v, &output_parameter)
		output_parameter.Id = k.(string)
		//fmt.Printf("C")
		new_array = append(new_array, output_parameter)
		//fmt.Printf("D")

	}
	//spew.Dump(new_array)
	return
}
