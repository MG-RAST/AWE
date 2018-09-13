package cwl

import (
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

//"fmt"
//"github.com/davecgh/go-spew/spew"
//"reflect"

// http://www.commonwl.org/v1.0/Workflow.html#ExpressionTool
type ExpressionTool struct {
	CWL_object_Impl `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
	CWL_class_Impl  `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
	CWL_id_Impl     `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
	Inputs          []InputParameter                `yaml:"inputs" bson:"inputs" json:"inputs" mapstructure:"inputs"`
	Outputs         []ExpressionToolOutputParameter `yaml:"outputs" bson:"outputs" json:"outputs" mapstructure:"outputs"`
	Expression      Expression                      `yaml:"expression,omitempty" bson:"expression,omitempty" json:"expression,omitempty" mapstructure:"expression,omitempty"`
	Requirements    []Requirement                   `yaml:"requirements,omitempty" bson:"requirements,omitempty" json:"requirements,omitempty" mapstructure:"requirements,omitempty"`
	Hints           []Requirement                   `yaml:"hints,omitempty" bson:"hints,omitempty" json:"hints,omitempty" mapstructure:"hints,omitempty"`
	Label           string                          `yaml:"label,omitempty" bson:"label,omitempty" json:"label,omitempty" mapstructure:"label,omitempty"`
	Doc             string                          `yaml:"doc,omitempty" bson:"doc,omitempty" json:"doc,omitempty" mapstructure:"doc,omitempty"`
	CwlVersion      CWLVersion                      `yaml:"cwlVersion,omitempty" bson:"cwlVersion,omitempty" json:"cwlVersion,omitempty" mapstructure:"cwlVersion,omitempty"`
	Namespaces      map[string]string               `yaml:"$namespaces,omitempty" bson:"_DOLLAR_namespaces,omitempty" json:"$namespaces,omitempty" mapstructure:"$namespaces,omitempty"`
}

// TODO pass along workflow InlineJavascriptRequirement
func NewExpressionTool(original interface{}, schemata []CWLType_Type, injectedRequirements []Requirement, context *WorkflowContext) (et *ExpressionTool, err error) {

	object, ok := original.(map[string]interface{})
	if !ok {
		err = fmt.Errorf("other types than map[string]interface{} not supported yet (got %s)", reflect.TypeOf(original))
		return
	}

	et = &ExpressionTool{}

	var inputs []InputParameter
	inputs_if, has_inputs := object["inputs"]
	if has_inputs {
		inputs, err = NewInputParameterArray(inputs_if, schemata)
		if err != nil {
			err = fmt.Errorf("(NewExpressionTool) error in NewInputParameterArray: %s", err.Error())
			return
		}
		object["inputs"] = inputs
	}

	outputs, has_outputs := object["outputs"]
	if has_outputs {
		object["outputs"], err = NewExpressionToolOutputParameterArray(outputs, schemata)
		if err != nil {
			err = fmt.Errorf("(NewExpressionTool) error in NewExpressionToolOutputParameterArray: %s", err.Error())
			return
		}
	}

	requirements, ok := object["requirements"]
	if !ok {
		requirements = nil
	}

	var requirements_array []Requirement
	//var requirements_array_temp *[]Requirement
	var schemata_new []CWLType_Type
	requirements_array, schemata_new, err = CreateRequirementArrayAndInject(requirements, injectedRequirements, inputs, context)
	if err != nil {
		err = fmt.Errorf("(NewExpressionTool) error in CreateRequirementArray (requirements): %s", err.Error())
		return
	}

	for i, _ := range schemata_new {
		schemata = append(schemata, schemata_new[i])
	}

	object["requirements"] = requirements_array

	hints, ok := object["hints"]
	if ok && (hints != nil) {
		var schemata_new []CWLType_Type

		var hints_array []Requirement
		hints_array, schemata, err = CreateHintsArray(hints, injectedRequirements, inputs, context)
		if err != nil {
			err = fmt.Errorf("(NewCommandLineTool) error in CreateRequirementArray (hints): %s", err.Error())
			return
		}
		for i, _ := range schemata_new {
			schemata = append(schemata, schemata_new[i])
		}
		object["hints"] = hints_array
	}

	err = mapstructure.Decode(object, et)
	if err != nil {
		err = fmt.Errorf("(NewExpressionTool) error parsing ExpressionTool class: %s", err.Error())
		return
	}
	if context.Namespaces != nil {
		et.Namespaces = context.Namespaces
	}
	if et.CwlVersion == "" {
		et.CwlVersion = context.CwlVersion
	}

	if et.CwlVersion == "" {
		err = fmt.Errorf("(NewExpressionTool) CwlVersion is empty !!!")
		return
	}

	var new_requirements []Requirement
	new_requirements, err = AddRequirement(NewInlineJavascriptRequirement(), et.Requirements)
	if err == nil {
		et.Requirements = new_requirements
	}

	return
}

func (et *ExpressionTool) Evaluate(inputs interface{}) (err error) {

	for i, _ := range et.Requirements {

		r := et.Requirements[i]

		err = r.Evaluate(inputs)
		if err != nil {
			err = fmt.Errorf("(ExpressionTool/Evaluate) Requirements r.Evaluate returned: %s", err.Error())
			return
		}

	}

	for i, _ := range et.Hints {

		r := et.Hints[i]

		err = r.Evaluate(inputs)
		if err != nil {
			err = fmt.Errorf("(ExpressionTool/Evaluate) Hints r.Evaluate returned: %s", err.Error())
			return
		}

	}

	return
}
