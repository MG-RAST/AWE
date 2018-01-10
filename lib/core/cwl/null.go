package cwl

import (
//"fmt"
//"strconv"
)

type Null string

func (i *Null) Is_CWL_object() {}

func (i *Null) GetClass() string      { return string(CWL_null) } // for CWL_object
func (i *Null) GetType() CWLType_Type { return CWL_null }
func (i *Null) String() string        { return "Null" }

func (i *Null) GetId() string  { return "" } // TODO deprecated functions
func (i *Null) SetId(x string) {}

func (i *Null) Is_CWL_minimal() {}

func NewNull() (n *Null) {

	var null_nptr Null
	null_nptr = Null("Null")

	n = &null_nptr

	return
}
