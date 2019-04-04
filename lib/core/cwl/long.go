package cwl

import (
	"fmt"
	"strconv"
)

type Long int64

func (i *Long) IsCWLObject() {}

func (i *Long) GetClass() string      { return string(CWLLong) } // for CWLObject
func (i *Long) GetType() CWLType_Type { return CWLLong }
func (i *Long) String() string        { return strconv.FormatInt(int64(*i), 10) }

func (i *Long) GetID() string  { return "" }
func (i *Long) SetID(x string) {}

func (i *Long) IsCWLMinimal() {}

func NewLong(value int64) (i *Long) {

	var i_nptr Long
	i_nptr = Long(value)

	i = &i_nptr

	return

}

func NewLongFromInterface(id string, native interface{}) (i *Long, err error) {

	_ = id

	real_int64, ok := native.(int64)
	if !ok {
		err = fmt.Errorf("(NewIntFromInterface) Cannot create int64")
		return
	}
	i = NewLong(real_int64)
	return
}
