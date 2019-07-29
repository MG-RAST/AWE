package cwl

import (
	"fmt"
	"strconv"
)

// Boolean _
type Boolean bool

// IsCWLObject _
func (b *Boolean) IsCWLObject() {}

// GetClass _
func (b *Boolean) GetClass() string { return string(CWLBoolean) } // for CWLObject

// GetType _
func (b *Boolean) GetType() CWLType_Type { return CWLBoolean }
func (b *Boolean) String() string        { return strconv.FormatBool(bool(*b)) }

// GetID _
func (b *Boolean) GetID() string { return "" }

// SetID _
func (b *Boolean) SetID(i string) {}

// IsCWLMinimal _
func (b *Boolean) IsCWLMinimal() {}

// NewBooleanFrombool _
func NewBooleanFrombool(value bool) (b *Boolean) {

	var bNptr Boolean
	bNptr = Boolean(value)

	b = &bNptr

	return

}

// NewBoolean _
func NewBoolean(id string, value bool) (b *Boolean) {

	_ = id

	return NewBooleanFrombool(value)

}

// NewBooleanFromInterface _
func NewBooleanFromInterface(id string, native interface{}) (b *Boolean, err error) {

	_ = id

	realBool, ok := native.(bool)
	if !ok {
		err = fmt.Errorf("(NewBooleanFromInterface) Cannot create bool")
		return
	}
	b = NewBooleanFrombool(realBool)
	return
}
