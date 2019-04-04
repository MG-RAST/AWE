package cwl

import (
	"fmt"
	//"github.com/mitchellh/mapstructure"
	"reflect"
	"strconv"
)

//type String struct {
//	CWLType_Impl `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
//	//Class        CWLType_Type `yaml:"class,omitempty" json:"class,omitempty" bson:"class,omitempty"`
//	Value string `yaml:"value,omitempty" json:"value,omitempty" bson:"value,omitempty"`
//}

type Number string

func (s *Number) GetClass() string      { return string(CWLString) } // for CWLObject
func (s *Number) GetType() CWLType_Type { return CWLString }
func (s *Number) String() string        { return string(*s) }

func (s *Number) GetID() string  { return "" }
func (s *Number) SetID(i string) {}

func (s *Number) IsCWLMinimal() {}

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

	switch native.(type) {
	case string:
		real_string, ok := native.(string)
		if !ok {
			err = fmt.Errorf("(NewStringFromInterface) Cannot create string")
			return
		}
		s = NewNumberFromstring(real_string)
	case float64:
		real_float64, _ := native.(float64)

		real_string := strconv.FormatFloat(real_float64, 'f', 6, 64)
		s = NewNumberFromstring(real_string)

	default:
		err = fmt.Errorf("(NewNumberFromInterface) type %s unknown", reflect.TypeOf(native))

	}
	return
}
