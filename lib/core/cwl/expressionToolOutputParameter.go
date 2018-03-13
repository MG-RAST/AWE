package cwl

// http://www.commonwl.org/v1.0/Workflow.html#ExpressionToolOutputParameter
type ExpressionToolOutputParameter struct {
	OutputParameter `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
}
