package cwl

import (
	"fmt"
	"reflect"
	//"github.com/davecgh/go-spew/spew"
)

type Array struct {
	CWLType_Impl `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // Class, Id, Type
	Items        []CWLType                                                             `yaml:"items,omitempty" json:"items,omitempty" bson:"items,omitempty"` // CWLType | CommandInputRecordSchema | CommandInputEnumSchema | CommandInputArraySchema | string | array<CWLType | CommandInputRecordSchema | CommandInputEnumSchema | CommandInputArraySchema | string>
	Items_Type   CWLType_Type                                                          `yaml:"items_type,omitempty" json:"items_type,omitempty" bson:"items_type,omitempty"`
}

//func (c *Array) GetClass() string { return "array" }

func (c *Array) Is_CWL_minimal()                {}
func (c *Array) Is_CommandInputParameterType()  {}
func (c *Array) Is_CommandOutputParameterType() {}

func NewArray(id string, native interface{}) (array *Array, err error) {

	//array = &Array{CWLType_Impl: CWLType_Impl{Id: id}}
	array = &Array{}
	array.Class = "array"
	//array.Type = CWLType_Type("array")
	array.Id = id

	if id != "" {
		array.Id = id
	}

	switch native.(type) {
	case map[string]interface{}:
		native_map := native.(map[string]interface{})

		items, has_items := native_map["items"]
		if has_items {
			var item_array *[]CWLType
			item_array, err = NewCWLTypeArray(items)
			if err != nil {
				err = fmt.Errorf("(NewArray) NewCWLTypeArray failed: %s", err.Error())
				return
			}

			array.Items = *item_array
			a_type := array.Items[0]
			array.Items_Type = a_type.GetType()
		}

	case []interface{}:
		native_array := native.([]interface{})
		for _, value := range native_array {

			value_cwl, xerr := NewCWLType("", value)
			if xerr != nil {
				err = xerr
				return
			}

			array.Items = append(array.Items, value_cwl)
		}
		if len(array.Items) > 0 {
			array.Items_Type = array.Items[0].GetType()
		}
		return
	default:
		err = fmt.Errorf("(NewArray) type %s unknown", reflect.TypeOf(native))
		return
	}

	return
}

func (c *Array) String() string {
	return "an array (TODO implement this)"
}

func (c *Array) Add(ct CWLType) {
	c.Items = append(c.Items, ct)
}
