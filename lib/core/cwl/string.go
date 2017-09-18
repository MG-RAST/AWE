package cwl

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
)

type String struct {
	CWLType_Impl `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
	//Class        CWLType_Type `yaml:"class,omitempty" json:"class,omitempty" bson:"class,omitempty"`
	Value string `yaml:"value,omitempty" json:"value,omitempty" bson:"value,omitempty"`
}

func (s *String) GetClass() string { return string(CWL_string) } // for CWL_object

func (s *String) String() string { return s.Value }

func NewString(id string, value string) (s *String) {

	s = &String{}
	s.Id = id
	s.Class = string(CWL_string)
	s.Type = CWL_string
	s.Value = value

	return

}

func NewStringFromInterface(id string, native interface{}) (s *String, err error) {
	//s = &String{Class: CWL_string}
	s = &String{}

	err = mapstructure.Decode(native, s)
	if err != nil {
		err = fmt.Errorf("(NewStringFromInterface) Could not convert fo string object")
		return
	}

	s.Class = string(CWL_string)
	s.Type = CWL_string

	if id != "" {
		s.Id = id
	}

	return
}
