package cwl

//"fmt"
//"github.com/davecgh/go-spew/spew"
//"reflect"

// http://www.commonwl.org/v1.0/Workflow.html#ExpressionTool
type ExpressionTool struct {
	CWL_object_Impl `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
	CWL_class_Impl  `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
	CWL_id_Impl     `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
	Inputs          []InputParameter                `yaml:"inputs,omitempty" bson:"inputs,omitempty" json:"inputs,omitempty" mapstructure:"inputs,omitempty"`
	Outputs         []ExpressionToolOutputParameter `yaml:"outputs,omitempty" bson:"outputs,omitempty" json:"outputs,omitempty" mapstructure:"outputs,omitempty"`
	Expression      Expression                      `yaml:"expression,omitempty" bson:"expression,omitempty" json:"expression,omitempty" mapstructure:"expression,omitempty"`
	Requirements    []Requirement                   `yaml:"requirements,omitempty" bson:"requirements,omitempty" json:"requirements,omitempty" mapstructure:"requirements,omitempty"`
	Hints           []Requirement                   `yaml:"hints,omitempty" bson:"hints,omitempty" json:"hints,omitempty" mapstructure:"hints,omitempty"`
	Label           string                          `yaml:"label,omitempty" bson:"label,omitempty" json:"label,omitempty" mapstructure:"label,omitempty"`
	Doc             string                          `yaml:"doc,omitempty" bson:"doc,omitempty" json:"doc,omitempty" mapstructure:"doc,omitempty"`
	CwlVersion      CWLVersion                      `yaml:"cwlVersion,omitempty" bson:"cwlVersion,omitempty" json:"cwlVersion,omitempty" mapstructure:"cwlVersion,omitempty"`
}
