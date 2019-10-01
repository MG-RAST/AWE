package cwl

import (
	"fmt"
	"strconv"
)

// Int _
type Int int

// IsCWLObject _
func (i *Int) IsCWLObject() {}

// GetClass _
func (i *Int) GetClass() string { return string(CWLInt) } // for CWLObject

// GetType _
func (i *Int) GetType() CWLType_Type { return CWLInt }
func (i *Int) String() string        { return strconv.Itoa(int(*i)) }

// GetID _
func (i *Int) GetID() string { return "" }

// SetID _
func (i *Int) SetID(x string) {}

// IsCWLMinimal _
func (i *Int) IsCWLMinimal() {}

// NewInt _
func NewInt(value int, context *WorkflowContext) (i *Int, err error) {

	var i_nptr Int
	i_nptr = Int(value)

	i = &i_nptr

	if context != nil && context.Initialzing {
		err = context.AddObject("", i, "NewInt")
		if err != nil {
			err = fmt.Errorf("(NewInt) context.Add returned: %s", err.Error())
			return
		}
	}

	return

}

// NewIntFromInterface _
func NewIntFromInterface(id string, native interface{}, context *WorkflowContext) (i *Int, err error) {

	_ = id

	realInt, ok := native.(int)
	if !ok {
		err = fmt.Errorf("(NewIntFromInterface) Cannot create int")
		return
	}
	i, err = NewInt(realInt, context)
	if err != nil {
		err = fmt.Errorf("(NewCWLType) NewInt: %s", err.Error())
		return
	}
	return
}
