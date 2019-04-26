package cwl

import (
	"fmt"
	"strconv"
)

type Double float64

func (i *Double) IsCWLObject() {}

func (i *Double) GetClass() string      { return string(CWLDouble) } // for CWLObject
func (i *Double) GetType() CWLType_Type { return CWLDouble }
func (i *Double) String() string        { return strconv.FormatFloat(float64(*i), 'f', -1, 64) }

func (i *Double) GetID() string  { return "" }
func (i *Double) SetID(x string) {}

func (i *Double) IsCWLMinimal() {}

func NewDoubleFromfloat64(value float64) (i *Double) {

	var i_nptr Double
	i_nptr = Double(value)

	i = &i_nptr

	return

}
func NewDouble(value float64) *Double {

	return NewDoubleFromfloat64(value)

}

func NewDoubleFromInterface(native interface{}) (i *Double, err error) {

	real_float64, ok := native.(float64)
	if !ok {
		err = fmt.Errorf("(NewDoubleFromInterface) Cannot create float64")
		return
	}
	i = NewDoubleFromfloat64(real_float64)
	return
}
