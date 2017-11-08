package cwl

import (
//"fmt"
//"strconv"
)

type Null int

func (i *Null) GetClass() string      { return string(CWL_null) } // for CWL_object
func (i *Null) GetType() CWLType_Type { return CWL_null }
func (i *Null) String() string        { return "null" }

func (i *Null) GetId() string  { return "" } // TODO deprecated functions
func (i *Null) SetId(x string) {}

func (i *Null) Is_CWL_minimal() {}

func NewNull() *Null {

	var something Null

	return &something
}
