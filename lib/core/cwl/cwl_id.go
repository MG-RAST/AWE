package cwl

import (
//"fmt"
//"reflect"
)

type CWL_id interface {
	//CWL_bbject_interface
	//GetClass() string
	GetID() string
	SetId(string)
	//is_Any()
}

type CWL_id_Impl struct {
	Id string `yaml:"id,omitempty" json:"id,omitempty" bson:"id,omitempty"`
}

func (c *CWL_id_Impl) GetID() string   { return c.Id }
func (c *CWL_id_Impl) SetId(id string) { c.Id = id; return }
