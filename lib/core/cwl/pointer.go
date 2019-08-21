package cwl

import (
	"fmt"
	"path"
	"strings"
)

// Pointer _
type Pointer string

//func (s *Pointer) GetClass() string      { return string(CWLPointer) } // for CWLObject
//func (s *Pointer) GetType() CWLType_Type { return CWLPointer }
//func (s *Pointer) String() string        { return string(*s) }

// GetID _
func (p *Pointer) GetID() string { return "" }

//func (s *Pointer) SetID(i string) {}

// Is_Type _
func (p *Pointer) Is_Type() {}

// Type2String _
func (p *Pointer) Type2String() string { return string(CWLPointer) }

//func (s *Pointer) IsCWLMinimal() {}

// NewPointerFromstring _
func NewPointerFromstring(value string) (s *Pointer) {

	var sNptr Pointer

	valueArray := strings.Split(value, "#")

	value = valueArray[len(valueArray)-1]
	value = path.Base(value)

	sNptr = Pointer(value)

	s = &sNptr

	return

}

// NewPointer _
func NewPointer(id string, value string) (s *Pointer) {

	_ = id

	return NewPointerFromstring(value)

}

// NewPointerFromInterface _
func NewPointerFromInterface(id string, native interface{}) (s *Pointer, err error) {

	_ = id

	realString, ok := native.(string)
	if !ok {
		err = fmt.Errorf("(NewPointerFromInterface) Cannot create string")
		return
	}
	s = NewPointerFromstring(realString)

	return
}
