package cwl

import (
	//"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
)

type CommandLineTool struct {
	Id                 string                   `yaml:"id,omitempty" bson:"id,omitempty" json:"id,omitempty"`
	Class              CWLType_Type             `yaml:"class,omitempty" bson:"class,omitempty" json:"class,omitempty"`
	BaseCommand        []string                 `yaml:"baseCommand,omitempty" bson:"baseCommand,omitempty" json:"baseCommand,omitempty"` // TODO also allow []string
	Inputs             []CommandInputParameter  `yaml:"inputs,omitempty" bson:"inputs,omitempty" json:"inputs,omitempty"`
	Outputs            []CommandOutputParameter `yaml:"outputs,omitempty" bson:"outputs,omitempty" json:"outputs,omitempty"`
	Hints              []Requirement            `yaml:"hints,omitempty" bson:"hints,omitempty" json:"hints,omitempty"` // TODO Any
	Label              string                   `yaml:"label,omitempty" bson:"label,omitempty" json:"label,omitempty"`
	Description        string                   `yaml:"description,omitempty" bson:"description,omitempty" json:"description,omitempty"`
	CwlVersion         CWLVersion               `yaml:"cwlVersion,omitempty" bson:"cwlVersion,omitempty" json:"cwlVersion,omitempty"`
	Arguments          []CommandLineBinding     `yaml:"arguments,omitempty" bson:"arguments,omitempty" json:"arguments,omitempty"` // TODO support CommandLineBinding
	Stdin              string                   `yaml:"stdin,omitempty" bson:"stdin,omitempty" json:"stdin,omitempty"`             // TODO support Expression
	Stdout             string                   `yaml:"stdout,omitempty" bson:"stdout,omitempty" json:"stdout,omitempty"`          // TODO support Expression
	SuccessCodes       []int                    `yaml:"successCodes,omitempty" bson:"successCodes,omitempty" json:"successCodes,omitempty"`
	TemporaryFailCodes []int                    `yaml:"temporaryFailCodes,omitempty" bson:"temporaryFailCodes,omitempty" json:"temporaryFailCodes,omitempty"`
	PermanentFailCodes []int                    `yaml:"permanentFailCodes,omitempty" bson:"permanentFailCodes,omitempty" json:"permanentFailCodes,omitempty"`
}

var CWL_CommandLineTool CWLType_Type = CWLType_Type("CommandLineTool")

func (c *CommandLineTool) GetClass() CWLType_Type {
	return CWL_CommandLineTool
}
func (c *CommandLineTool) GetId() string   { return c.Id }
func (c *CommandLineTool) SetId(id string) { c.Id = id }
func (c *CommandLineTool) Is_CWL_minimal() {}
func (c *CommandLineTool) Is_process()     {}

// keyname will be converted into 'Id'-field

//func NewCommandLineTool(object CWL_object_generic) (commandLineTool *CommandLineTool, err error) {
func NewCommandLineTool(generic interface{}) (commandLineTool *CommandLineTool, err error) {

	fmt.Println("NewCommandLineTool:")
	spew.Dump(generic)

	//switch type()
	object, ok := generic.(map[string]interface{})
	if !ok {
		err = fmt.Errorf("other types than map[string]interface{} not supported yet")
		return
	}

	commandLineTool = &CommandLineTool{Class: CWL_CommandLineTool}

	inputs, ok := object["inputs"]
	if ok {
		// Convert map of inputs into array of inputs
		err, object["inputs"] = CreateCommandInputArray(inputs)
		if err != nil {
			err = fmt.Errorf("(NewCommandLineTool) error in CreateCommandInputArray: %s", err.Error())
			return
		}
	}
	outputs, ok := object["outputs"]
	if ok {
		// Convert map of outputs into array of outputs
		object["outputs"], err = NewCommandOutputParameterArray(outputs)
		if err != nil {
			err = fmt.Errorf("(NewCommandLineTool) error in NewCommandOutputParameterArray: %s", err.Error())
			return
		}
	}

	baseCommand, ok := object["baseCommand"]
	if ok {
		object["baseCommand"], err = NewBaseCommandArray(baseCommand)
		if err != nil {
			err = fmt.Errorf("(NewCommandLineTool) error in NewBaseCommandArray: %s", err.Error())
			return
		}
	}

	arguments, ok := object["arguments"]
	if ok {
		// Convert map of outputs into array of outputs
		object["arguments"], err = NewCommandLineBindingArray(arguments)
		if err != nil {
			err = fmt.Errorf("(NewCommandLineTool) error in NewCommandLineBindingArray: %s", err.Error())
			return
		}
	}

	//switch object["hints"].(type) {
	//case map[interface{}]interface{}:
	hints, ok := object["hints"]
	if ok {
		object["hints"], err = CreateRequirementArray(hints)
		if err != nil {
			err = fmt.Errorf("(NewCommandLineTool) error in CreateRequirementArray: %s", err.Error())
			return
		}
	}
	//}

	err = mapstructure.Decode(object, commandLineTool)
	if err != nil {
		err = fmt.Errorf("(NewCommandLineTool) error parsing CommandLineTool class: %s", err.Error())
		return
	}
	spew.Dump(commandLineTool)

	return
}

func NewBaseCommandArray(original interface{}) (new_array []string, err error) {
	switch original.(type) {
	case []interface{}:
		for _, v := range original.([]interface{}) {

			v_str, ok := v.(string)
			if !ok {
				err = fmt.Errorf("(NewBaseCommandArray) []interface{} array element is not a string")
				return
			}
			new_array = append(new_array, v_str)
		}

		return
	case string:
		org_str, _ := original.(string)
		new_array = append(new_array, org_str)
		return
	default:
		err = fmt.Errorf("(NewBaseCommandArray) type unknown")
		return
	}
	return
}
