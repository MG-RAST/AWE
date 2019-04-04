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

type String string

func (s *String) IsCWLObject() {}

func (s *String) GetClass() string      { return string(CWL_string) } // for CWLObject
func (s *String) GetType() CWLType_Type { return CWL_string }
func (s *String) String() string        { return string(*s) }

func (s *String) GetID() string  { return "" }
func (s *String) SetId(i string) {}

func (s *String) Is_CWL_minimal() {}

func NewString(value string) (s *String) {

	var s_nptr String
	s_nptr = String(value)

	s = &s_nptr

	return

}

func NewStringFromInterface(native interface{}) (s *String, err error) {

	real_string, ok := native.(string)
	if !ok {
		err = fmt.Errorf("(NewStringFromInterface) Cannot create string")
		return
	}
	s = NewString(real_string)

	return
}
