package cwl

import "fmt"

// used for ExpressionToolOutputParameter, WorkflowOutputParameter, CommandOutputParameter
type OutputParameter struct {
	Id             string                `yaml:"id,omitempty" bson:"id,omitempty" json:"id,omitempty"`
	Label          string                `yaml:"label,omitempty" bson:"label,omitempty" json:"label,omitempty"`
	SecondaryFiles []Expression          `yaml:"secondaryFiles,omitempty" bson:"secondaryFiles,omitempty" json:"secondaryFiles,omitempty"` // TODO string | Expression | array<string | Expression>
	Format         Expression            `yaml:"format,omitempty" bson:"format,omitempty" json:"format,omitempty"`
	Streamable     bool                  `yaml:"streamable,omitempty" bson:"streamable,omitempty" json:"streamable,omitempty"`
	OutputBinding  *CommandOutputBinding `yaml:"outputBinding,omitempty" bson:"outputBinding,omitempty" json:"outputBinding,omitempty"`
	Type           []interface{}         `yaml:"type,omitempty" bson:"type,omitempty" json:"type,omitempty"`
}

func NormalizeOutputParameter(original_map map[string]interface{}) (err error) {

	outputBinding, ok := original_map["outputBinding"]
	if ok {
		original_map["outputBinding"], err = NewCommandOutputBinding(outputBinding)
		if err != nil {
			err = fmt.Errorf("(NewCommandOutputParameter) NewCommandOutputBinding returns %s", err.Error())
			return
		}
	}

	return
}
