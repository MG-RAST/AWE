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
	coas = &CommandOutputArraySchema{}

	switch original.(type) {
	case map[interface{}]interface{}:
		original_map, ok := original.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("type error")
			return
		}
		return NewCommandOutputArraySchema(original_map)
	case map[string]interface{}:
		original_map, ok := original.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("type error")
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
