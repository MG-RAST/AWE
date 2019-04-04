package cwl

import (
	"fmt"
)

type Pointer string

//func (s *Pointer) GetClass() string      { return string(CWL_pointer) } // for CWLObject
//func (s *Pointer) GetType() CWLType_Type { return CWL_pointer }
//func (s *Pointer) String() string        { return string(*s) }

func (s *Pointer) GetID() string { return "" }

//func (s *Pointer) SetId(i string) {}
func (c *Pointer) Is_Type()            {}
func (c *Pointer) Type2String() string { return string(CWL_pointer) }

//func (s *Pointer) Is_CWL_minimal() {}

func NewPointerFromstring(value string) (s *Pointer) {

	var s_nptr Pointer
	s_nptr = Pointer(value)

	s = &s_nptr

	return

}

func NewPointer(id string, value string) (s *Pointer) {

	_ = id

	return NewPointerFromstring(value)

}

func NewPointerFromInterface(id string, native interface{}) (s *Pointer, err error) {

	_ = id

	real_string, ok := native.(string)
	if !ok {
		err = fmt.Errorf("(NewPointerFromInterface) Cannot create string")
		return
	}
	s = NewPointerFromstring(real_string)

	return
}
