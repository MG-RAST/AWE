package cwl

import (
	"fmt"
	//"github.com/mitchellh/mapstructure"
	"strconv"
)

type Boolean bool

func (b *Boolean) GetClass() string      { return string(CWL_boolean) } // for CWL_object
func (b *Boolean) GetType() CWLType_Type { return CWL_boolean }
func (b *Boolean) String() string        { return strconv.FormatBool(bool(*b)) }

func (b *Boolean) GetId() string  { return "" }
func (b *Boolean) SetId(i string) {}

func (b *Boolean) Is_CWL_minimal() {}

//type Boolean struct {
//	CWLType_Impl `bson:",inline" json:",inline" mapstructure:",squash"`
//	Value        bool `yaml:"value,omitempty" json:"value,omitempty" bson:"value,omitempty"`
//}

//func (s *Boolean) GetClass() string      { return string(CWL_boolean) } // for CWL_object
//func (s *Boolean) GetType() CWLType_Type { return CWL_boolean }
//func (s *Boolean) String() string {
//	if s.Value {
//		return "True"
//	}
//	return "False"
//}

//func (s *Boolean) Is_CommandInputParameterType() {} // for CommandInputParameterType

func NewBooleanFrombool(value bool) (b *Boolean) {

	var b_nptr Boolean
	b_nptr = Boolean(value)

	b = &b_nptr

	return

}
func NewBoolean(id string, value bool) (b *Boolean) {

	_ = id

	return NewBooleanFrombool(value)

}

func NewBooleanFromInterface(id string, native interface{}) (b *Boolean, err error) {

	_ = id

	real_bool, ok := native.(bool)
	if !ok {
		err = fmt.Errorf("(NewBooleanFromInterface) Cannot create bool")
		return
	}
	b = NewBooleanFrombool(real_bool)
	return
}

// func NewBoolean(id string, value bool) *Boolean {
// 	b := &Boolean{}
// 	b.Class = "boolean"
// 	b.Type = CWL_boolean
// 	b.Id = id
// 	b.Value = value
// 	return b
// }
//
// func NewBooleanFromInterface(id string, original interface{}) (b *Boolean, err error) {
// 	b = &Boolean{}
//
// 	b.Class = "boolean"
// 	b.Type = CWL_boolean
// 	b.Id = id
//
// 	err = mapstructure.Decode(original, b)
// 	if err != nil {
// 		err = fmt.Errorf("(NewBooleanFromInterface) %s", err.Error())
// 		return
// 	}
//
// 	return
// }
