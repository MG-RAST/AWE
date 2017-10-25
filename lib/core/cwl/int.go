package cwl

import (
	"fmt"
	"strconv"
)

type Int int

func (i *Int) GetClass() string      { return string(CWL_int) } // for CWL_object
func (i *Int) GetType() CWLType_Type { return CWL_int }
func (i *Int) String() string        { return strconv.Itoa(int(*i)) }

func (i *Int) GetId() string  { return "" }
func (i *Int) SetId(x string) {}

func (i *Int) Is_CWL_minimal() {}

func NewIntFromint(value int) (i *Int) {

	var i_nptr Int
	i_nptr = Int(value)

	i = &i_nptr

	return

}
func NewInt(id string, value int) *Int {

	_ = id

	return NewIntFromint(value)

}

func NewIntFromInterface(id string, native interface{}) (i *Int, err error) {

	_ = id

	real_int, ok := native.(int)
	if !ok {
		err = fmt.Errorf("(NewIntFromInterface) Cannot create int")
		return
	}
	i = NewIntFromint(real_int)
	return
}
