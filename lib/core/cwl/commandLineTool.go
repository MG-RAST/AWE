package cwl

import (
	//"errors"
	"fmt"
	//"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
)

// http://www.commonwl.org/v1.0/CommandLineTool.html#CommandLineTool
type CommandLineTool struct {
	//Id                 string                   `yaml:"id,omitempty" bson:"id,omitempty" json:"id,omitempty"`
	//Class              string                   `yaml:"class,omitempty" bson:"class,omitempty" json:"class,omitempty"`
	CWL_object_Impl    `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
	BaseCommand        []string                 `yaml:"baseCommand,omitempty" bson:"baseCommand,omitempty" json:"baseCommand,omitempty" mapstructure:"baseCommand,omitempty"`
	Inputs             []CommandInputParameter  `yaml:"inputs,omitempty" bson:"inputs,omitempty" json:"inputs,omitempty" mapstructure:"inputs,omitempty"`
	Outputs            []CommandOutputParameter `yaml:"outputs,omitempty" bson:"outputs,omitempty" json:"outputs,omitempty" mapstructure:"outputs,omitempty"`
	Hints              []Requirement            `yaml:"hints,omitempty" bson:"hints,omitempty" json:"hints,omitempty mapstructure:"hints,omitempty""`
	Requirements       []Requirement            `yaml:"requirements,omitempty" bson:"requirements,omitempty" json:"requirements,omitempty" mapstructure:"requirements,omitempty"`
	Doc                string                   `yaml:"doc,omitempty" bson:"doc,omitempty" json:"doc,omitempty" mapstructure:"doc,omitempty"`
	Label              string                   `yaml:"label,omitempty" bson:"label,omitempty" json:"label,omitempty" mapstructure:"label,omitempty"`
	Description        string                   `yaml:"description,omitempty" bson:"description,omitempty" json:"description,omitempty" mapstructure:"description,omitempty"`
	CwlVersion         CWLVersion               `yaml:"cwlVersion,omitempty" bson:"cwlVersion,omitempty" json:"cwlVersion,omitempty" mapstructure:"cwlVersion,omitempty"`
	Arguments          []CommandLineBinding     `yaml:"arguments,omitempty" bson:"arguments,omitempty" json:"arguments,omitempty" mapstructure:"arguments,omitempty"`
	Stdin              string                   `yaml:"stdin,omitempty" bson:"stdin,omitempty" json:"stdin,omitempty" mapstructure:"stdin,omitempty"`     // TODO support Expression
	Stdout             string                   `yaml:"stdout,omitempty" bson:"stdout,omitempty" json:"stdout,omitempty" mapstructure:"stdout,omitempty"` // TODO support Expression
	SuccessCodes       []int                    `yaml:"successCodes,omitempty" bson:"successCodes,omitempty" json:"successCodes,omitempty" mapstructure:"successCodes,omitempty"`
	TemporaryFailCodes []int                    `yaml:"temporaryFailCodes,omitempty" bson:"temporaryFailCodes,omitempty" json:"temporaryFailCodes,omitempty" mapstructure:"temporaryFailCodes,omitempty"`
	PermanentFailCodes []int                    `yaml:"permanentFailCodes,omitempty" bson:"permanentFailCodes,omitempty" json:"permanentFailCodes,omitempty" mapstructure:"permanentFailCodes,omitempty"`
}

func (c *CommandLineTool) Is_CWL_minimal() {}
func (c *CommandLineTool) Is_process()     {}

// keyname will be converted into 'Id'-field

//func NewCommandLineTool(object CWL_object_generic) (commandLineTool *CommandLineTool, err error) {
func NewCommandLineTool(generic interface{}) (commandLineTool *CommandLineTool, err error) {

	//fmt.Println("NewCommandLineTool:")
	//spew.Dump(generic)

	//switch type()
	object, ok := generic.(map[string]interface{})
	if !ok {
		err = fmt.Errorf("other types than map[string]interface{} not supported yet")
		return
	}

	commandLineTool = &CommandLineTool{}
	commandLineTool.Class = "CommandLineTool"
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

	arguments, has_arguments := object["arguments"]
	if has_arguments {
		// Convert map of outputs into array of outputs
		var arguments_object []CommandLineBinding
		arguments_object, err = NewCommandLineBindingArray(arguments)
		if err != nil {
			err = fmt.Errorf("(NewCommandLineTool) error in NewCommandLineBindingArray: %s", err.Error())
			return
		}
		//delete(object, "arguments")
		object["arguments"] = arguments_object
	}

	//switch object["hints"].(type) {
	//case map[interface{}]interface{}:
	hints, ok := object["hints"]
	if ok {
		object["hints"], err = CreateRequirementArray(hints)
		if err != nil {
			err = fmt.Errorf("(NewCommandLineTool) error in CreateRequirementArray (hints): %s", err.Error())
			return
		}
	}

	requirements, ok := object["requirements"]
	if ok {
		object["requirements"], err = CreateRequirementArray(requirements)
		if err != nil {
			err = fmt.Errorf("(NewCommandLineTool) error in CreateRequirementArray (requirements): %s", err.Error())
			return
		}
	}

	//}
	//id, _ := object["id"]
	//if id == "#blat.tool.cwl" {
	//delete(object, "inputs")
	//delete(object, "outputs")
	//delete(object, "baseCommand")
	//delete(object, "arguments")
	//delete(object, "hints")
	//delete(object, "requirements")
	//	fmt.Println("after thinning out")
	//	spew.Dump(object)
	//}

	err = mapstructure.Decode(object, commandLineTool)
	if err != nil {
		err = fmt.Errorf("(NewCommandLineTool) error parsing CommandLineTool class: %s", err.Error())
		return
	}
	//if has_arguments {
	//	object["arguments"] = arguments_object // mapstructure.Decode has some issues, no idea why
	//}
	//spew.Dump(commandLineTool)
	//if id == "#blat.tool.cwl" {
	//	panic("done")
	//}
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
