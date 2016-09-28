package cwl

import (
	//"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
)

type CommandLineTool struct {
	Id                 string                   `yaml:"id"`
	BaseCommand        string                   `yaml:"baseCommand"` // TODO also allow []string
	Inputs             []CommandInputParameter  `yaml:"inputs"`
	Outputs            []CommandOutputParameter `yaml:"outputs"`
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

func (c *CommandLineTool) GetClass() string { return "CommandLineTool" }
func (c *CommandLineTool) GetId() string    { return c.Id }
func (c *CommandLineTool) SetId(id string)  { c.Id = id }

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
	Id             string               `yaml:"id"`
	SecondaryFiles []Expression         `yaml:"secondaryFiles"` // TODO string | Expression | array<string | Expression>
	Format         string               `yaml:"format"`
	Streamable     bool                 `yaml:"streamable"`
	Type           string               `yaml:"type"` // TODO CWLType | CommandInputRecordSchema | CommandInputEnumSchema | CommandInputArraySchema | string | array<CWLType | CommandInputRecordSchema | CommandInputEnumSchema | CommandInputArraySchema | string>
	Label          string               `yaml:"label"`
	Description    string               `yaml:"description"`
	OutputBinding  CommandOutputBinding `yaml:"outputBinding"`
}

// keyname will be converted into 'Id'-field
func CreateCommandInputArray(original interface{}) (err error, new_array []CommandInputParameter) {

	for k, v := range original.(map[interface{}]interface{}) {

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

func getGlob(object interface{}) (expressions []Expression, err error) {

	for key_if, value_if := range object.(map[interface{}]interface{}) {

		key_str, ok := key_if.(string)
		if ok {
			fmt.Printf("key_str: %s\n", key_str)
			if key_str == "glob" {
				switch value_if.(type) {
				case string:
					expression_str := value_if.(string)
					expression := Expression(expression_str)
					expressions = []Expression{expression}
				case []string:
					expressions = value_if.([]Expression)
				default:
					err = fmt.Errorf("cannot parse glob")
				}
				return
			}
		}
	}
	return
}

func getCommandOutputBinding(object interface{}) (outputBinding CommandOutputBinding, err error) {

	// find "outputBinding" in object

	for key_if, value_if := range object.(map[interface{}]interface{}) {
		key_str, ok := key_if.(string)
		if ok {
			fmt.Printf("key_str: %s\n", key_str)
			if key_str == "outputBinding" {

				//var outputBinding CommandOutputBinding
				mapstructure.Decode(value_if, &outputBinding)
				//output_parameter.OutputBinding = outputBinding

				outputBinding.Glob, err = getGlob(value_if)

				return
			}
		} else {
			fmt.Printf("not ok\n")
		}
	}
	return
}

// keyname will be converted into 'Id'-field
func CreateCommandOutputArray(original interface{}) (err error, new_array []CommandOutputParameter) {

	for id_if, output_parameter_if := range original.(map[interface{}]interface{}) {
		fmt.Printf("AAAAAAAAA")
		spew.Dump(output_parameter_if)

		var output_parameter CommandOutputParameter
		mapstructure.Decode(output_parameter_if, &output_parameter)

		outputBinding, xerr := getCommandOutputBinding(output_parameter_if)
		if xerr != nil {
			err = xerr
			return
		}
		output_parameter.OutputBinding = outputBinding

		output_parameter.Id = id_if.(string)
		spew.Dump(output_parameter)

		new_array = append(new_array, output_parameter)
		//fmt.Printf("D")

	}
	//spew.Dump(new_array)
	return
}

func getCommandLineTool(object CWL_object_generic) (commandLineTool *CommandLineTool, err error) {

	commandLineTool = &CommandLineTool{}

	switch object["inputs"].(type) {
	case map[interface{}]interface{}:
		// Convert map of inputs into array of inputs
		err, object["inputs"] = CreateCommandInputArray(object["inputs"])
		if err != nil {
			return
		}
	}

	switch object["outputs"].(type) {
	case map[interface{}]interface{}:
		// Convert map of outputs into array of outputs
		err, object["outputs"] = CreateCommandOutputArray(object["outputs"])
		if err != nil {
			return
		}
	}

	err = mapstructure.Decode(object, commandLineTool)
	if err != nil {
		err = fmt.Errorf("error parsing CommandLineTool class: %s", err.Error())
		return
	}
	//spew.Dump(commandLineTool)

	return
}
