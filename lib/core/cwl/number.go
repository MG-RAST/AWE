package cwl

import (
	"fmt"
	//"github.com/mitchellh/mapstructure"
)

//type String struct {
//	CWLType_Impl `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
//	//Class        CWLType_Type `yaml:"class,omitempty" json:"class,omitempty" bson:"class,omitempty"`
//	Value string `yaml:"value,omitempty" json:"value,omitempty" bson:"value,omitempty"`
//}

type Number string

func (s *Number) GetClass() string      { return string(CWL_string) } // for CWL_object
func (s *Number) GetType() CWLType_Type { return CWL_string }
func (s *Number) String() string        { return string(*s) }

func (s *Number) GetId() string  { return "" }
func (s *Number) SetId(i string) {}

func (s *Number) Is_CWL_minimal() {}

func NewNumberFromstring(value string) (s *Number) {

	var s_nptr Number
	s_nptr = Number(value)

	s = &s_nptr

	return

}

func NewNumber(id string, value string) (s *Number) {

	_ = id

	return NewNumberFromstring(value)

}

func NewNumberFromInterface(id string, native interface{}) (s *Number, err error) {

	_ = id

	real_string, ok := native.(string)
	if !ok {
		err = fmt.Errorf("(NewStringFromInterface) Cannot create string")
		return
	}
	s = NewNumberFromstring(real_string)

	return
}
