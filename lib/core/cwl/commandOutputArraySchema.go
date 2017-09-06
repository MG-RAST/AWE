package cwl

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
	"reflect"
)

type CommandOutputArraySchema struct {
	Items         []string              `yaml:"items,omitempty" bson:"items,omitempty" json:"items,omitempty"`
	Type          string                `yaml:"type,omitempty" bson:"type,omitempty" json:"type,omitempty"` // must be array
	Label         string                `yaml:"label,omitempty" bson:"label,omitempty" json:"label,omitempty"`
	OutputBinding *CommandOutputBinding `yaml:"outputBinding,omitempty" bson:"outputBinding,omitempty" json:"outputBinding,omitempty"`
}

func (c *CommandOutputArraySchema) Is_CommandOutputParameterType() {}

func NewCommandOutputArraySchema(original interface{}) (coas *CommandOutputArraySchema, err error) {

	original, err = makeStringMap(original)
	if err != nil {
		return
	}

	coas = &CommandOutputArraySchema{}
	coas.Type = "array"
	switch original.(type) {

	case map[string]interface{}:
		original_map, ok := original.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewCommandOutputArraySchema) type error b")
			return
		}

		items, ok := original_map["items"]
		if ok {
			items_string, ok := items.(string)
			if ok {
				original_map["items"] = []string{items_string}
			}
		}

		err = mapstructure.Decode(original, coas)
		if err != nil {
			err = fmt.Errorf("(NewCommandOutputArraySchema) %s", err.Error())
			return
		}
	default:
		err = fmt.Errorf("NewCommandOutputArraySchema, unknown type %s", reflect.TypeOf(original))
	}
	return
}
