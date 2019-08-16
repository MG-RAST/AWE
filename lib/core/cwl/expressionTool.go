package cwl

import (
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

//"fmt"
//"github.com/davecgh/go-spew/spew"
//"reflect"

// ExpressionTool http://www.commonwl.org/v1.0/Workflow.html#ExpressionTool
type ExpressionTool struct {
	ProcessImpl `yaml:",inline" bson:",inline" json:",inline" mapstructure:"-"`
	Inputs      []InputParameter       `yaml:"inputs" bson:"inputs" json:"inputs" mapstructure:"inputs"`
	Outputs     map[string]interface{} `yaml:"outputs" bson:"outputs" json:"outputs" mapstructure:"outputs"` // ExpressionToolOutputParameter
	Expression  Expression             `yaml:"expression,omitempty" bson:"expression,omitempty" json:"expression,omitempty" mapstructure:"expression,omitempty"`
	Label       string                 `yaml:"label,omitempty" bson:"label,omitempty" json:"label,omitempty" mapstructure:"label,omitempty"`
	Doc         string                 `yaml:"doc,omitempty" bson:"doc,omitempty" json:"doc,omitempty" mapstructure:"doc,omitempty"`
	CwlVersion  CWLVersion             `yaml:"cwlVersion,omitempty" bson:"cwlVersion,omitempty" json:"cwlVersion,omitempty" mapstructure:"cwlVersion,omitempty"`
	Namespaces  map[string]string      `yaml:"$namespaces,omitempty" bson:"_DOLLAR_namespaces,omitempty" json:"$namespaces,omitempty" mapstructure:"$namespaces,omitempty"`
}

// NewExpressionTool TODO pass along workflow InlineJavascriptRequirement
func NewExpressionTool(original interface{}, parentIdentifier string, objectIdentifier string, schemata []CWLType_Type, injectedRequirements []Requirement, context *WorkflowContext) (et *ExpressionTool, err error) {

	object, ok := original.(map[string]interface{})
	if !ok {
		err = fmt.Errorf("other types than map[string]interface{} not supported yet (got %s)", reflect.TypeOf(original))
		return
	}

	et = &ExpressionTool{}

	et.ProcessImpl = ProcessImpl{}
	var process *ProcessImpl
	process = &et.ProcessImpl
	process.Class = "ExpressionTool"
	err = ProcessImplInit(original, process, parentIdentifier, objectIdentifier, context)
	if err != nil {
		err = fmt.Errorf("(NewExpressionTool) NewProcessImpl returned: %s", err.Error())
		return
	}

	et.ProcessImpl = *process

	var inputs []InputParameter
	inputsIf, hasInputs := object["inputs"]
	if hasInputs {
		inputs, err = NewInputParameterArray(inputsIf, schemata, context)
		if err != nil {
			err = fmt.Errorf("(NewExpressionTool) error in NewInputParameterArray: %s", err.Error())
			return
		}
		object["inputs"] = inputs
	}

	outputs, hasOutputs := object["outputs"]
	if hasOutputs {
		object["outputs"], err = NewExpressionToolOutputParameterMap(outputs, schemata, context)
		if err != nil {
			err = fmt.Errorf("(NewExpressionTool) error in NewExpressionToolOutputParameterMap: %s", err.Error())
			return
		}
	}
	err = CreateRequirementAndHints(object, process, injectedRequirements, inputs, context)
	if err != nil {
		err = fmt.Errorf("(NewExpressionTool) CreateRequirementArrayAndInject returned: %s", err.Error())
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
		err = fmt.Errorf("(NewExpressionTool) CwlVersion is empty !!! ")
		return
	}

	var newRequirements []Requirement
	ijr := NewInlineJavascriptRequirement()

	newRequirements, err = AddRequirement(&ijr, et.Requirements)
	if err == nil {
		et.Requirements = newRequirements
	}

	// if context != nil && err == nil {
	// 	err = context.Add(et.Id, et, "NewExpressionTool")
	// 	if err != nil {
	// 		err = fmt.Errorf("(NewExpressionTool) context.Add returned: %s", err.Error())
	// 		return
	// 	}
	// }

	return
}

func (et *ExpressionTool) Evaluate(inputs interface{}, context *WorkflowContext) (err error) {

	for i, _ := range et.Requirements {

		r := et.Requirements[i]

		err = r.Evaluate(inputs, context)
		if err != nil {
			err = fmt.Errorf("(ExpressionTool/Evaluate) Requirements r.Evaluate returned: %s", err.Error())
			return
		}

	}

	for i, _ := range et.Hints {

		r := et.Hints[i]

		err = r.Evaluate(inputs, context)
		if err != nil {
			err = fmt.Errorf("(ExpressionTool/Evaluate) Hints r.Evaluate returned: %s", err.Error())
			return
		}

	}

	return
}
