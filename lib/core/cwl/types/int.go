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
	return &Int{CWLType_Impl: CWLType_Impl{Id: id}, Value: value}
}

func (i *Int) GetClass() string { return CWL_int }

func (i *Int) String() string { return strconv.Itoa(i.Value) }
