package cwl

import (
	"fmt"
	"strconv"
)

type Boolean bool

func (b *Boolean) IsCWLObject() {}

func (b *Boolean) GetClass() string      { return string(CWLBoolean) } // for CWLObject
func (b *Boolean) GetType() CWLType_Type { return CWLBoolean }
func (b *Boolean) String() string        { return strconv.FormatBool(bool(*b)) }

func (b *Boolean) GetID() string  { return "" }
func (b *Boolean) SetID(i string) {}

func (b *Boolean) IsCWLMinimal() {}

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
