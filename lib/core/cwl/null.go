package cwl

import (
//"fmt"
//"strconv"
)

type Null struct {
	CWLType_Impl `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
}

func NewNull() *Null {
	n := &Null{}
	n.Class = "null"
	n.Type = CWL_null
	return n
}

func NewNullWithId(id string) *Null {
	n := NewNull()
	n.Id = id
	return n
}

func (n *Null) GetClass() string { return string(CWL_null) }
func (n *Null) String() string   { return "null" }
