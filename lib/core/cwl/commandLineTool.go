package cwl

import (
	//"errors"
	"fmt"
	"path"
	"strings"

	//"github.com/davecgh/go-spew/spew"
	"reflect"

	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
)

// http://www.commonwl.org/v1.0/CommandLineTool.html#CommandLineTool
type CommandLineTool struct {
	//Id                 string                   `yaml:"id,omitempty" bson:"id,omitempty" json:"id,omitempty"`
	//Class              string                   `yaml:"class,omitempty" bson:"class,omitempty" json:"class,omitempty"`
	CWLObjectImpl      `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
	CWL_class_Impl     `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
	IdentifierImpl     `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
	ProcessImpl        `yaml:",inline" bson:",inline" json:",inline" mapstructure:",squash"`
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
	Namespaces         map[string]string        `yaml:"$namespaces,omitempty" bson:"_DOLLAR_namespaces,omitempty" json:"$namespaces,omitempty" mapstructure:"$namespaces,omitempty"`
}

// IsCWLMinimal _
func (c *CommandLineTool) IsCWLMinimal() {}

// IsProcess _
func (c *CommandLineTool) IsProcess() {}

// keyname will be converted into 'Id'-field

// NewCommandLineTool _
// parentIdentifier is used to convert relative id to absolute id
// objectIdentifier is used when there is no local is, in case of file or embedded tool
func NewCommandLineTool(generic interface{}, parentIdentifier string, objectIdentifier string, injectedRequirements []Requirement, context *WorkflowContext) (commandLineTool *CommandLineTool, schemata []CWLType_Type, err error) {

	//fmt.Println("NewCommandLineTool() generic:")
	//spew.Dump(generic)

	//switch type()
	object, ok := generic.(map[string]interface{})
	if !ok {
		err = fmt.Errorf("other types than map[string]interface{} not supported yet (got %s)", reflect.TypeOf(generic))
		return
	}
	//spew.Dump(generic)

	//fmt.Println("NewCommandLineTool() object:")
	//spew.Dump(object)

	commandLineTool = &CommandLineTool{}
	commandLineTool.Class = "CommandLineTool"

	//fmt.Printf("(NewCommandLineTool) requirements %d\n", len(requirements_array))
	//spew.Dump(requirements_array)
	//scs := spew.ConfigState{Indent: "\t"}
	//scs.Dump(object["requirements"])

	if objectIdentifier != "" {
		object["id"] = objectIdentifier
	} else {

		objectIDIf, hasID := object["id"]
		if hasID {

			objectID := objectIDIf.(string)

			if !strings.HasPrefix(objectID, "#") {
				if parentIdentifier == "" {
					err = fmt.Errorf("(NewCommandLineTool) parentIdentifier is needed but empty, objectID=%s", objectID)
					return
				}
				objectID = path.Join(parentIdentifier, objectID)
			}

			if !strings.HasPrefix(objectID, "#") {
				err = fmt.Errorf("(NewCommandLineTool) not absolute: objectID=%s , parentIdentifier=%s, objectIdentifier=%s", objectID, parentIdentifier, objectIdentifier)
				return
			}
			object["id"] = objectID
		}
	}

	// extract SchemaDefRequirement
	var schemaDefReq *SchemaDefRequirement
	//var schemataNew []CWLType_Type
	hasSchemaDefReq := false

	requirementsIf, hasRequirements := object["requirements"]
	if hasRequirements {
		schemaDefReq, hasSchemaDefReq, err = GetSchemaDefRequirement(requirementsIf, context)
		if err != nil {
			err = fmt.Errorf("(NewCommandLineTool) GetSchemaDefRequirement returned: %s", err.Error())
			return
		}
	}

	//if has_schema_def_req {
	//	for i, _ := range schemataNew {
	//		schemata = append(schemata, schemataNew[i])
	//	}
	//}

	inputs := []*CommandInputParameter{}

	inputsIf, ok := object["inputs"]
	if ok {
		// Convert map of inputs into array of inputs
		err, inputs = CreateCommandInputArray(inputsIf, schemata, context)
		if err != nil {
			err = fmt.Errorf("(NewCommandLineTool) error in CreateCommandInputArray: %s", err.Error())
			return
		}
	}

	object["inputs"] = inputs

	var copa []interface{}

	outputs, hasOutputs := object["outputs"]
	if hasOutputs {

		//fmt.Println("NewCommandLineTool() object/outputs:")
		//spew.Dump(outputs)

		// Convert map of outputs into array of outputs

		copa, err = NewCommandOutputParameterArray(outputs, schemata, context)
		if err != nil {
			//fmt.Println("NewCommandLineTool after error")
			//spew.Dump(object)

			err = fmt.Errorf("(NewCommandLineTool) error in NewCommandOutputParameterArray: %s", err.Error())
			return
		}
		object["outputs"] = copa
	} else {
		err = fmt.Errorf("(NewCommandLineTool) no outputs !?")
		return
		//object["outputs"] = []*CommandOutputParameter{}
	}

	baseCommand, ok := object["baseCommand"]
	if ok {
		object["baseCommand"], err = NewBaseCommandArray(baseCommand)
		if err != nil {
			err = fmt.Errorf("(NewCommandLineTool) error in NewBaseCommandArray: %s", err.Error())
			return
		}
	}

	arguments, hasArguments := object["arguments"]
	if hasArguments {
		// Convert map of outputs into array of outputs
		var argumentsObject []CommandLineBinding
		argumentsObject, err = NewCommandLineBindingArray(arguments, context)
		if err != nil {
			err = fmt.Errorf("(NewCommandLineTool) error in NewCommandLineBindingArray: %s", err.Error())
			return
		}
		//delete(object, "arguments")
		object["arguments"] = argumentsObject
	}

	if hasSchemaDefReq {
		injectedRequirements = append(injectedRequirements, schemaDefReq)
	}

	var requirementsArray []Requirement

	//fmt.Printf("(NewCommandLineTool) Injecting %d\n", len(requirementsArray))
	//spew.Dump(requirementsArray)
	requirementsArray, err = CreateRequirementArrayAndInject(requirementsIf, injectedRequirements, nil, context)
	if err != nil {
		err = fmt.Errorf("(NewCommandLineTool) error in CreateRequirementArray (requirements): %s", err.Error())
		return
	}

	//for i, _ := range schemataNew {
	//	schemata = append(schemata, schemataNew[i])
	//}

	object["requirements"] = requirementsArray

	hints, ok := object["hints"]
	if ok && (hints != nil) {
		//var schemataNew []CWLType_Type

		var hintsArray []Requirement
		hintsArray, err = CreateHintsArray(hints, injectedRequirements, inputs, context)
		if err != nil {
			err = fmt.Errorf("(NewCommandLineTool) error in CreateRequirementArray (hints): %s", err.Error())
			return
		}
		//for i, _ := range schemataNew {
		//	schemata = append(schemata, schemataNew[i])
		//}
		object["hints"] = hintsArray
	}

	err = mapstructure.Decode(object, commandLineTool)
	if err != nil {
		err = fmt.Errorf("(NewCommandLineTool) error parsing CommandLineTool class: %s", err.Error())
		return
	}

	if commandLineTool.ID == "" {
		err = fmt.Errorf("(NewCommandLineTool) id is empty!?")
		return
	}

	if !strings.HasPrefix(commandLineTool.ID, "#") {
		err = fmt.Errorf("(NewCommandLineTool) id is not absolute!?")
		return
	}

	//fmt.Println("commandLineTool:")
	//spew.Dump(commandLineTool)

	if commandLineTool.CwlVersion == "" {
		commandLineTool.CwlVersion = context.CwlVersion
	}

	if context == nil {
		err = fmt.Errorf("(NewCommandLineTool) context == nil")
		return
	}

	if context.Namespaces != nil {
		commandLineTool.Namespaces = context.Namespaces
	}

	if context != nil {

		thisID := commandLineTool.ID
		if thisID == "" {
			err = fmt.Errorf("(NewCommandLineTool) did not expect empty Id")
			return
		}

		// err = context.Add(commandLineTool.Id, commandLineTool, "NewCommandLineTool")
		// if err != nil {
		// 	err = fmt.Errorf("(NewCommandLineTool) (add commandLineTool) context.Add returned: %s", err.Error())
		// 	return
		// }

		// for i := range commandLineTool.Inputs {
		// 	inp := &commandLineTool.Inputs[i]
		// 	inpID := path.Join(thisID, inp.Id)

		// 	err = context.AddObject(inpID, inp, "NewCommandLineTool")
		// 	if err != nil {
		// 		err = fmt.Errorf("(NewCommandLineTool) X (add commandLineToolInput)  context.Add returned: %s", err.Error())
		// 		return
		// 	}
		// }
	}
	return

}

// NewBaseCommandArray _
func NewBaseCommandArray(original interface{}) (newArray []string, err error) {
	newArray = []string{}
	switch original.(type) {
	case []interface{}:
		originalArray := original.([]interface{})
		for _, v := range originalArray {

			vStr, ok := v.(string)
			if !ok {
				spew.Dump(originalArray)
				err = fmt.Errorf("(NewBaseCommandArray) []interface{} array element is not a string, it is %s", reflect.TypeOf(v))
				return
			}
			newArray = append(newArray, vStr)
		}

		return
	case string:
		orgStr, _ := original.(string)
		newArray = append(newArray, orgStr)
		return
	default:
		err = fmt.Errorf("(NewBaseCommandArray) type unknown")

	}
	return
}

// Evaluate _
func (c *CommandLineTool) Evaluate(inputs interface{}, context *WorkflowContext) (err error) {

	for i := range c.Requirements {

		r := c.Requirements[i]

		err = r.Evaluate(inputs, context)
		if err != nil {
			err = fmt.Errorf("(CommandLineTool/Evaluate) Requirements r.Evaluate returned: %s", err.Error())
			return
		}

	}

	for i := range c.Hints {

		r := c.Hints[i]

		err = r.Evaluate(inputs, context)
		if err != nil {
			err = fmt.Errorf("(CommandLineTool/Evaluate) Hints r.Evaluate returned: %s", err.Error())
			return
		}

	}

	return
}
