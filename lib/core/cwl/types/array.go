package cwl

import (
//"fmt"
//"github.com/davecgh/go-spew/spew"
)

type Array struct {
	CWL_object
	Id         string
	Items      []CWLType
	Items_Type string
}

func (c *Array) GetClass() string { return CWL_array }
func (c *Array) GetId() string    { return c.Id }
func (c *Array) SetId(id string)  { c.Id = id }

func (c *Array) Is_CWL_minimal()                {}
func (c *Array) Is_CWLType()                    {}
func (c *Array) Is_CommandInputParameterType()  {}
func (c *Array) Is_CommandOutputParameterType() {}

func NewArray(id string, native []interface{}) (array *Array, err error) {

	array = &Array{}

	if id != "" {
		array.Id = id
	}

	for _, value := range native {

		value_cwl, xerr := NewCWLType("", value)
		if xerr != nil {
			err = xerr
			return
		}

		array.Items = append(array.Items, value_cwl)
	}
	if len(array.Items) > 0 {
		array.Items_Type = array.Items[0].GetClass()
	}

	return
}
