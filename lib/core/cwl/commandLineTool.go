package cwl

import (
	//"errors"
	"fmt"
	//"github.com/davecgh/go-spew/spew"
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
func (c *CommandLineTool) is_CWL_minimal()  {}

// keyname will be converted into 'Id'-field

func NewCommandLineTool(object CWL_object_generic) (commandLineTool *CommandLineTool, err error) {

	commandLineTool = &CommandLineTool{}

	inputs, ok := object["inputs"]
	if ok {
		// Convert map of inputs into array of inputs
		err, object["inputs"] = CreateCommandInputArray(inputs)
		if err != nil {
			err = fmt.Errorf("error in CreateCommandInputArray: %s", err.Error())
			return
		}
	}
	outputs, ok := object["outputs"]
	if ok {
		// Convert map of outputs into array of outputs
		object["outputs"], err = NewCommandOutputParameterArray(outputs)
		if err != nil {
			err = fmt.Errorf("error in NewCommandOutputParameterArray: %s", err.Error())
			return
		}
	}

	//switch object["hints"].(type) {
	//case map[interface{}]interface{}:
	hints, ok := object["hints"]
	if ok {
		object["hints"], err = CreateRequirementArray(hints)
		if err != nil {
			err = fmt.Errorf("error in CreateRequirementArray: %s", err.Error())
			return
		}
	}
	//}

	err = mapstructure.Decode(object, commandLineTool)
	if err != nil {
		err = fmt.Errorf("error parsing CommandLineTool class: %s", err.Error())
		return
	}
	//spew.Dump(commandLineTool)

	return
}
