package cwl

import (
//"fmt"
//"reflect"
)

type CWL_class interface {
	//CWL_minimal_interface
	GetClass() string
}

type CWL_class_Impl struct {
	//Id    string `yaml:"id,omitempty" json:"id,omitempty" bson:"id,omitempty"`
	Class string `yaml:"class,omitempty" json:"class,omitempty" bson:"class,omitempty"`
}

func (c *CWL_class_Impl) GetClass() string { return c.Class }
