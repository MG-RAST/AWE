package cwl

import (
	"fmt"
	//"reflect"
	"encoding/json"
	"github.com/davecgh/go-spew/spew"
)

type Array []CWLType

//type Array struct {
//CWLType_Impl `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // Class, Id, Type
//	Items []CWLType `yaml:"items,omitempty" json:"items,omitempty" bson:"items,omitempty"` // CWLType | CommandInputRecordSchema | CommandInputEnumSchema | CommandInputArraySchema | string | array<CWLType | CommandInputRecordSchema | CommandInputEnumSchema | CommandInputArraySchema | string>
//Items_Type   CWLType_Type                                                          `yaml:"items_type,omitempty" json:"items_type,omitempty" bson:"items_type,omitempty"`
//}

func (c *Array) Is_CWL_object() {}

func (c *Array) GetClass() string      { return "array" }
func (c *Array) GetId() string         { return "foobar" }
func (c *Array) SetId(string)          {}
func (c *Array) GetType() CWLType_Type { return CWL_array }

func (c *Array) Is_CWL_minimal() {}

//func (c *Array) Is_CommandInputParameterType()  {}
//func (c *Array) Is_CommandOutputParameterType() {}

func NewArray(id string, native interface{}) (array_ptr *Array, err error) {

	switch native.(type) {
	case []interface{}:
		native_array := native.([]interface{})

		array := Array{}
		for _, value := range native_array {

			var value_cwl CWLType
			value_cwl, err = NewCWLType("", value)
			if err != nil {
				fmt.Println("NewArray element:")
				spew.Dump(value)

				err = fmt.Errorf("(NewArray) NewCWLType returned: %s", err.Error())
				return
			}

			array = append(array, value_cwl)
		}
		//if len(array.Items) > 0 {
		//	array.Items_Type = array.Items[0].GetType()
		//}
		array_ptr = &array
		return
	default:
	}

	return
}

//func (c *Array) GetClass() string { return c.Class }

// func NewArray(id string, native interface{}) (array *Array, err error) {
//
// 	//array = &Array{CWLType_Impl: CWLType_Impl{Id: id}}
// 	//array = &Array{}
// 	//array.Class = "array"
// 	//schema := NewArraySchema()
// 	//array.Type = CWL_array
// 	//array.Type =
//
// 	//if id != "" {
// 	//		array.Id = id
// 	//	}
//
// 	switch native.(type) {
// 	case map[string]interface{}:
// 		native_map := native.(map[string]interface{})
//
// 		obj_id, has_id := native_map["id"]
// 		if has_id {
// 			obj_id_str, ok := obj_id.(string)
// 			if !ok {
// 				err = fmt.Errorf("(NewArray) id is not of type string")
// 				return
// 			}
//
// 			array.Id = obj_id_str
// 		}
//
// 		items, has_items := native_map["items"]
// 		if has_items {
// 			var item_array *[]CWLType
// 			item_array, err = NewCWLTypeArray(items, array.Id)
// 			if err != nil {
// 				err = fmt.Errorf("(NewArray) NewCWLTypeArray failed: %s", err.Error())
// 				return
// 			}
//
// 			array.Items = *item_array
//
// 			//type_map := make(map[CWLType_Type]bool)
//
// 			// collect info about types used in the array, needed later for verification
// 			//for _, item := range array.Items {
//
// 			//	item_type := item.GetType()
// 			//	type_map[item_type] = true
//
// 			//}
//
// 			//for item_type, _ := range type_map {
// 			//	schema.Items = append(schema.Items, item_type)
// 			//}
//
// 			//a_type := array.Items[0]
// 			//array.Items_Type = a_type.GetType()
// 		}
//
// 	case []interface{}:
// 		native_array := native.([]interface{})
// 		for _, value := range native_array {
//
// 			value_cwl, xerr := NewCWLType("", value)
// 			if xerr != nil {
// 				err = xerr
// 				return
// 			}
//
// 			array.Items = append(array.Items, value_cwl)
// 		}
// 		//if len(array.Items) > 0 {
// 		//	array.Items_Type = array.Items[0].GetType()
// 		//}
// 		return
// 	default:
// 		err = fmt.Errorf("(NewArray) type %s unknown", reflect.TypeOf(native))
// 		return
// 	}
//
// 	return
// }
//
func (c *Array) String() string {

	a_byte, err := json.Marshal(c)
	if err != nil {
		return err.Error()
	}

	return string(a_byte[:])
}

//
// func (c *Array) Add(ct CWLType) {
// 	c.Items = append(c.Items, ct)
// }
