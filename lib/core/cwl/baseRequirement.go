package cwl

// BaseRequirement _
type BaseRequirement struct {
	CWLObjectImpl `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
	Class         string `yaml:"class,omitempty" json:"class,omitempty" bson:"class,omitempty" mapstructure:"class,omitempty"`
}

// GetClass _
func (c BaseRequirement) GetClass() string { return c.Class }

// Evaluate _
func (c BaseRequirement) Evaluate(inputs interface{}, context *WorkflowContext) (err error) { return }
