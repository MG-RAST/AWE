package cwl

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

type EnumSchema struct {
	Symbols []string `yaml:"symbols,omitempty" bson:"symbols,omitempty" json:"symbols,omitempty" mapstructure:"symbols,omitempty"`
	Type    string   `yaml:"type,omitempty" bson:"type,omitempty" json:"type,omitempty" mapstructure:"type,omitempty"` // must be enum
	Label   string   `yaml:"label,omitempty" bson:"label,omitempty" json:"label,omitempty" mapstructure:"label,omitempty"`
	Name    string   `yaml:"name,omitempty" bson:"name,omitempty" json:"name,omitempty" mapstructure:"name,omitempty"`
}

func (s EnumSchema) GetID() string       { return s.Name }
func (s EnumSchema) Is_Type()            {}
func (s EnumSchema) Type2String() string { return "EnumSchema" }

func NewEnumSchemaFromInterface(original interface{}, context *WorkflowContext) (es EnumSchema, err error) {

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	err = mapstructure.Decode(original, &es)
	if err != nil {
		err = fmt.Errorf("(NewEnumSchemaFromInterface) mapstructure returned: %s", err.Error())
		return
	}

	return
}
