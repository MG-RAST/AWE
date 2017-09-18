package cwl

import (
	//"fmt"
	"strconv"
)

type Int struct {
	CWLType_Impl `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
	Value        int `yaml:"value,omitempty" json:"value,omitempty" bson:"value,omitempty"`
}

func NewInt(id string, value int) *Int {
	i := &Int{}
	i.Id = id
	i.Class = "int"
	i.Type = CWL_int
	i.Value = value
	return i
}

func (i *Int) GetClass() string { return string(CWL_int) }

func (i *Int) String() string { return strconv.Itoa(i.Value) }
