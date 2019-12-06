package cwl

import (
	"fmt"
	"math"
	"strconv"
)

// Double _
type Double float64

// IsCWLObject _
func (i *Double) IsCWLObject() {}

// GetClass _
func (i *Double) GetClass() string { return string(CWLDouble) } // for CWLObject
// GetType _
func (i *Double) GetType() CWLType_Type { return CWLDouble }
func (i *Double) String() string        { return strconv.FormatFloat(float64(*i), 'f', -1, 64) }

// GetID _
func (i *Double) GetID() string { return "" }

// SetID _
func (i *Double) SetID(x string) {}

// IsCWLMinimal _
func (i *Double) IsCWLMinimal() {}

// NewDoubleFromfloat64 _
func NewDoubleFromfloat64(value float64) (i *Double) {

	var i_nptr Double
	i_nptr = Double(value)

	i = &i_nptr

	return

}

// NewDouble _
func NewDouble(value float64) *Double {

	return NewDoubleFromfloat64(value)

}

// NewDoubleFromInterface _
func NewDoubleFromInterface(native interface{}) (i *Double, err error) {

	real_float64, ok := native.(float64)
	if !ok {
		err = fmt.Errorf("(NewDoubleFromInterface) Cannot create float64")
		return
	}
	i = NewDoubleFromfloat64(real_float64)

	if math.IsNaN(real_float64) {
		err = fmt.Errorf("(NewDoubleFromInterface) not a number , NaN error")
	}

	return
}
