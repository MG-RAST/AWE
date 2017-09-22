package cwl

import (
//"fmt"
//"strconv"
)

type Null struct {
	CWLType_Impl `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
}

func NewNull(id string) *Null {
	n := &Null{}
	n.Id = id
	n.Class = "null"
	n.Type = CWL_null
	return n
}

func (n *Null) GetClass() string { return string(CWL_null) }
func (n *Null) String() string   { return "null" }
