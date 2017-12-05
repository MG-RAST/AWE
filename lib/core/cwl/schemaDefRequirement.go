package cwl

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
)

// http://www.commonwl.org/v1.0/Workflow.html#SchemaDefRequirement
type SchemaDefRequirement struct {
	BaseRequirement `bson:",inline" yaml:",inline" json:",inline"`
	Types           []interface{} `yaml:"types,omitempty" json:"types,omitempty" bson:"types,omitempty"` // array<InputRecordSchema | InputEnumSchema | InputArraySchema>
}

func (c SchemaDefRequirement) GetId() string { return "None" }

func NewSchemaDefRequirement(original interface{}) (r *SchemaDefRequirement, err error) {

	original_map, ok := original.(map[string]interface{})
	if !ok {
		err = fmt.Errorf("(NewSchemaDefRequirement) type error")
		return
	}

	types, has_types := original_map["types"]
	if has_types {

		var array []CWLType_Type
		array, err = NewCWLType_TypeArray(types, "SchemaDefRequirement") // TODO fix context
		if err != nil {
			return
		}
		original_map["types"] = array
	}

	var requirement SchemaDefRequirement
	r = &requirement

	err = mapstructure.Decode(original, &requirement)
	if err != nil {
		return
	}

	requirement.Class = "SchemaDefRequirement"

	return
}
