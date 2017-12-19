package cwl

import (
	"fmt"
)

type Enum string

//type Enum struct {
//	CWLType_Impl `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
//	Symbols      []string `yaml:"symbols,omitempty" json:"symbols,omitempty" bson:"symbols,omitempty"`
//}

func (s *Enum) GetClass() string      { return string(CWL_enum) } // for CWL_object
func (s *Enum) GetType() CWLType_Type { return CWL_enum }
func (s *Enum) String() string        { return string(*s) }

func (s *Enum) GetId() string  { return "" }
func (s *Enum) SetId(i string) {}

func (s *Enum) Is_CWL_minimal() {}

func NewEnumFromstring(value string) (s *Enum) {

	var s_nptr Enum
	s_nptr = Enum(value)

	s = &s_nptr

	return

}

func NewEnum(id string, value string) (s *Enum) {

	_ = id

	return NewEnumFromstring(value)

}

func NewEnumFromInterface(id string, native interface{}) (s *Enum, err error) {

	_ = id

	real_string, ok := native.(string)
	if !ok {
		err = fmt.Errorf("(NewEnumFromInterface) Cannot create string")
		return
	}
	s = NewEnumFromstring(real_string)

	return
}
