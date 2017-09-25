package cwl

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
)

type Boolean struct {
	CWLType_Impl `bson:",inline" json:",inline" mapstructure:",squash"`
	Value        bool `yaml:"value,omitempty" json:"value,omitempty" bson:"value,omitempty"`
}

func (s *Boolean) GetClass() string      { return string(CWL_boolean) } // for CWL_object
func (s *Boolean) GetType() CWLType_Type { return CWL_boolean }
func (s *Boolean) String() string {
	if s.Value {
		return "True"
	}
	return "False"
}

//func (s *Boolean) Is_CommandInputParameterType() {} // for CommandInputParameterType

func NewBoolean(id string, value bool) *Boolean {
	b := &Boolean{}
	b.Class = "boolean"
	b.Type = CWL_boolean
	b.Id = id
	b.Value = value
	return b
}

func NewBooleanFromInterface(id string, original interface{}) (b *Boolean, err error) {
	b = &Boolean{}

	b.Class = "boolean"
	b.Type = CWL_boolean
	b.Id = id

	err = mapstructure.Decode(original, b)
	if err != nil {
		err = fmt.Errorf("(NewBooleanFromInterface) %s", err.Error())
		return
	}

	return
}
