package cwl

import (
	//"errors"
	"fmt"
	//"github.com/davecgh/go-spew/spew"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

// http://www.commonwl.org/v1.0/CommandLineTool.html#CommandLineTool
type CommandLineTool struct {
	//Id                 string                   `yaml:"id,omitempty" bson:"id,omitempty" json:"id,omitempty"`
	//Class              string                   `yaml:"class,omitempty" bson:"class,omitempty" json:"class,omitempty"`
	CWL_object_Impl    `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
	CWL_class_Impl     `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
	CWL_id_Impl        `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
	BaseCommand        []string                 `yaml:"baseCommand,omitempty" bson:"baseCommand,omitempty" json:"baseCommand,omitempty" mapstructure:"baseCommand,omitempty"`
	Inputs             []CommandInputParameter  `yaml:"inputs" bson:"inputs" json:"inputs" mapstructure:"inputs"`
	Outputs            []CommandOutputParameter `yaml:"outputs" bson:"outputs" json:"outputs" mapstructure:"outputs"`
	Hints              []Requirement            `yaml:"hints,omitempty" bson:"hints,omitempty" json:"hints,omitempty" mapstructure:"hints,omitempty"`
	Requirements       []Requirement            `yaml:"requirements,omitempty" bson:"requirements,omitempty" json:"requirements,omitempty" mapstructure:"requirements,omitempty"`
	Doc                string                   `yaml:"doc,omitempty" bson:"doc,omitempty" json:"doc,omitempty" mapstructure:"doc,omitempty"`
	Label              string                   `yaml:"label,omitempty" bson:"label,omitempty" json:"label,omitempty" mapstructure:"label,omitempty"`
	Description        string                   `yaml:"description,omitempty" bson:"description,omitempty" json:"description,omitempty" mapstructure:"description,omitempty"`
	CwlVersion         CWLVersion               `yaml:"cwlVersion,omitempty" bson:"cwlVersion,omitempty" json:"cwlVersion,omitempty" mapstructure:"cwlVersion,omitempty"`
	Arguments          []CommandLineBinding     `yaml:"arguments,omitempty" bson:"arguments,omitempty" json:"arguments,omitempty" mapstructure:"arguments,omitempty"`
	Stdin              string                   `yaml:"stdin,omitempty" bson:"stdin,omitempty" json:"stdin,omitempty" mapstructure:"stdin,omitempty"`     // TODO support Expression
	Stderr             string                   `yaml:"stderr,omitempty" bson:"stderr,omitempty" json:"stderr,omitempty" mapstructure:"stderr,omitempty"` // TODO support Expression
	Stdout             string                   `yaml:"stdout,omitempty" bson:"stdout,omitempty" json:"stdout,omitempty" mapstructure:"stdout,omitempty"` // TODO support Expression
	SuccessCodes       []int                    `yaml:"successCodes,omitempty" bson:"successCodes,omitempty" json:"successCodes,omitempty" mapstructure:"successCodes,omitempty"`
	TemporaryFailCodes []int                    `yaml:"temporaryFailCodes,omitempty" bson:"temporaryFailCodes,omitempty" json:"temporaryFailCodes,omitempty" mapstructure:"temporaryFailCodes,omitempty"`
	PermanentFailCodes []int                    `yaml:"permanentFailCodes,omitempty" bson:"permanentFailCodes,omitempty" json:"permanentFailCodes,omitempty" mapstructure:"permanentFailCodes,omitempty"`
	Namespaces         map[string]string        `yaml:"namespaces,omitempty" bson:"namespaces,omitempty" json:"namespaces,omitempty" mapstructure:"namespaces,omitempty"`
}

func (c *CommandLineTool) Is_CWL_minimal() {}
func (c *CommandLineTool) Is_process()     {}

// keyname will be converted into 'Id'-field

//func NewCommandLineTool(object CWL_object_generic) (commandLineTool *CommandLineTool, err error) {
func NewCommandLineTool(generic interface{}, cwl_version CWLVersion, injectedRequirements []Requirement, namespaces map[string]string) (commandLineTool *CommandLineTool, schemata []CWLType_Type, err error) {

	//fmt.Println("NewCommandLineTool:")
	//spew.Dump(generic)

	//switch type()
	object, ok := generic.(map[string]interface{})
	if !ok {
		err = fmt.Errorf("other types than map[string]interface{} not supported yet (got %s)", reflect.TypeOf(generic))
		return
	}
	//spew.Dump(generic)

	commandLineTool = &CommandLineTool{}
	commandLineTool.Class = "CommandLineTool"

	//fmt.Printf("(NewCommandLineTool) requirements %d\n", len(requirements_array))
	//spew.Dump(requirements_array)
	//scs := spew.ConfigState{Indent: "\t"}
	//scs.Dump(object["requirements"])

	requirements, ok := object["requirements"]
	if !ok {
		requirements = nil
	}

	// extract SchemaDefRequirement
	var schema_def_req *SchemaDefRequirement
	var schemata_new []CWLType_Type
	has_schema_def_req := false

	if requirements != nil {
		schema_def_req, schemata_new, has_schema_def_req, err = GetSchemaDefRequirement(requirements)
		if err != nil {
			err = fmt.Errorf("(NewCommandLineTool) GetSchemaDefRequirement returned: %s", err.Error())
			return
		}
	}

	if has_schema_def_req {
		for i, _ := range schemata_new {
			schemata = append(schemata, schemata_new[i])
		}
	}

	inputs := []*CommandInputParameter{}

	inputs_if, ok := object["inputs"]
	if ok {
		// Convert map of inputs into array of inputs
		err, inputs = CreateCommandInputArray(inputs_if, schemata)
		if err != nil {
			err = fmt.Errorf("(NewCommandLineTool) error in CreateCommandInputArray: %s", err.Error())
			return
		}
	}

	object["inputs"] = inputs

	outputs, ok := object["outputs"]
	if ok {
		// Convert map of outputs into array of outputs
		object["outputs"], err = NewCommandOutputParameterArray(outputs, schemata)
		if err != nil {
			err = fmt.Errorf("(NewCommandLineTool) error in NewCommandOutputParameterArray: %s", err.Error())
			return
		}
	} else {
		object["outputs"] = []*CommandOutputParameter{}
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

	var requirements_array []Requirement

	//fmt.Printf("(NewCommandLineTool) Injecting %d\n", len(requirements_array))
	//spew.Dump(requirements_array)
	requirements_array, schemata_new, err = CreateRequirementArrayAndInject(requirements, injectedRequirements, nil)
	if err != nil {
		err = fmt.Errorf("(NewCommandLineTool) error in CreateRequirementArray (requirements): %s", err.Error())
		return
	}

	for i, _ := range schemata_new {
		schemata = append(schemata, schemata_new[i])
	}

	if has_schema_def_req {
		requirements_array = append(requirements_array, schema_def_req)
	}
	object["requirements"] = requirements_array

	hints, ok := object["hints"]
	if ok && (hints != nil) {
		var schemata_new []CWLType_Type

		var hints_array []Requirement
		hints_array, schemata, err = CreateHintsArray(hints, injectedRequirements, inputs)
		if err != nil {
			err = fmt.Errorf("(NewCommandLineTool) error in CreateRequirementArray (hints): %s", err.Error())
			return
		}
		for i, _ := range schemata_new {
			schemata = append(schemata, schemata_new[i])
		}
		object["hints"] = hints_array
	}

	err = mapstructure.Decode(object, commandLineTool)
	if err != nil {
		err = fmt.Errorf("(NewCommandLineTool) error parsing CommandLineTool class: %s", err.Error())
		return
	}

	if commandLineTool.CwlVersion == "" {
		commandLineTool.CwlVersion = cwl_version
	}

	if namespaces != nil {
		commandLineTool.Namespaces = namespaces
	}
	return

}

func NewBaseCommandArray(original interface{}) (new_array []string, err error) {
	new_array = []string{}
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

	}
	return
}

func (c *CommandLineTool) Evaluate(inputs interface{}) (err error) {

	for i, _ := range c.Requirements {

		r := c.Requirements[i]

		err = r.Evaluate(inputs)
		if err != nil {
			err = fmt.Errorf("(CommandLineTool/Evaluate) Requirements r.Evaluate returned: %s", err.Error())
			return
		}

	}

	for i, _ := range c.Hints {

		r := c.Hints[i]

		err = r.Evaluate(inputs)
		if err != nil {
			err = fmt.Errorf("(CommandLineTool/Evaluate) Hints r.Evaluate returned: %s", err.Error())
			return
		}

	}

	return
}
