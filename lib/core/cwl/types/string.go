package cwl

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
)

type String struct {
	CWLType_Impl `yaml:"-"`
	Id           string `yaml:"id,omitempty" json:"id,omitempty" bson:"id,omitempty"`
	Class        string `yaml:"class,omitempty" json:"class,omitempty" bson:"class,omitempty"`
	Value        string `yaml:"value,omitempty" json:"value,omitempty" bson:"value,omitempty"`
}

func (s *String) GetClass() string { return CWL_string } // for CWL_object
func (s *String) GetId() string    { return s.Id }       // for CWL_object
func (s *String) SetId(id string)  { s.Id = id }
func (s *String) String() string   { return s.Value }

func NewString(id string, value string) (s *String) {
	return &String{Class: CWL_string, Id: id, Value: value}
}

func NewStringFromInterface(native interface{}) (s *String, err error) {
	s = &String{Class: CWL_string}

	err = mapstructure.Decode(native, s)
	if err != nil {
		err = fmt.Errorf("(NewStringFromInterface) Could not convert fo string object")
		return
	}
	return
}
