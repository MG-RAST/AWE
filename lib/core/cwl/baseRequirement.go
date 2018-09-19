package cwl

type BaseRequirement struct {
	Class string `yaml:"class,omitempty" json:"class,omitempty" bson:"class,omitempty" mapstructure:"class,omitempty"`
}

func (c BaseRequirement) GetClass() string                                                  { return c.Class }
func (c BaseRequirement) Evaluate(inputs interface{}, context *WorkflowContext) (err error) { return }
