package cwl

import (
//"fmt"
//"strconv"
)

// YAML: null | Null | NULL | ~

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

// https://mlafeldt.github.io/blog/decoding-yaml-in-go/

// No, use this: https://godoc.org/gopkg.in/yaml.v2#Marshaler
func (n *Null) MarshalYAML() (i interface{}, err error) {

	i = nil

	return
}

func (n *Null) MarshalJSON() (b []byte, err error) {

	b = []byte("null")
	return
}
