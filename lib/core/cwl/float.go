package cwl

import (
	"fmt"
	"strconv"
)

type Float float32

func (i *Float) IsCWLObject() {}

func (i *Float) GetClass() string      { return string(CWLFloat) } // for CWLObject
func (i *Float) GetType() CWLType_Type { return CWLFloat }
func (i *Float) String() string        { return strconv.FormatFloat(float64(*i), 'f', -1, 32) }

func (i *Float) GetID() string  { return "" }
func (i *Float) SetID(x string) {}

func (i *Float) IsCWLMinimal() {}

func NewFloatFromfloat32(value float32) (i *Float) {

	var i_nptr Float
	i_nptr = Float(value)

	i = &i_nptr

	return

}
func NewFloat(value float32) *Float {

	return NewFloatFromfloat32(value)

}

func NewFloatFromInterface(native interface{}) (i *Float, err error) {

	real_float32, ok := native.(float32)
	if !ok {
		err = fmt.Errorf("(NewFloatFromInterface) Cannot create float32")
		return
	}
	i = NewFloatFromfloat32(real_float32)
	return
}
