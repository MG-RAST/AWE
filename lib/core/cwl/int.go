package cwl

import (
	"fmt"
	"strconv"
)

type Int int

func (i *Int) IsCWLObject() {}

func (i *Int) GetClass() string      { return string(CWLInt) } // for CWLObject
func (i *Int) GetType() CWLType_Type { return CWLInt }
func (i *Int) String() string        { return strconv.Itoa(int(*i)) }

func (i *Int) GetID() string  { return "" }
func (i *Int) SetID(x string) {}

func (i *Int) IsCWLMinimal() {}

func NewInt(value int, context *WorkflowContext) (i *Int, err error) {

	var i_nptr Int
	i_nptr = Int(value)

	i = &i_nptr

	if context != nil && context.Initialzing {
		err = context.Add("", i, "NewInt")
		if err != nil {
			err = fmt.Errorf("(NewInt) context.Add returned: %s", err.Error())
			return
		}
	}

	return

}

func NewIntFromInterface(id string, native interface{}, context *WorkflowContext) (i *Int, err error) {

	_ = id

	real_int, ok := native.(int)
	if !ok {
		err = fmt.Errorf("(NewIntFromInterface) Cannot create int")
		return
	}
	i, err = NewInt(real_int, context)
	if err != nil {
		err = fmt.Errorf("(NewCWLType) NewInt: %s", err.Error())
		return
	}
	return
}
