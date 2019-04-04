package cwl

import (
	"fmt"
	"reflect"
	//"reflect"
	"encoding/json"

	"github.com/davecgh/go-spew/spew"
)

type Array []CWLType

func (c *Array) Is_CWL_object() {}

func (c *Array) GetClass() string      { return "array" }
func (c *Array) GetID() string         { return "foobar" }
func (c *Array) SetId(string)          {}
func (c *Array) GetType() CWLType_Type { return CWL_array }

func (c *Array) Is_CWL_minimal() {}

//func (c *Array) Is_CommandInputParameterType()  {}
//func (c *Array) Is_CommandOutputParameterType() {}

func NewArray(id string, native interface{}, context *WorkflowContext) (array_ptr *Array, err error) {

	switch native.(type) {
	case []interface{}:
		native_array := native.([]interface{})

		array := Array{}
		for _, value := range native_array {

			var value_cwl CWLType
			value_cwl, err = NewCWLType("", value, context)
			if err != nil {
				fmt.Println("NewArray element:")
				spew.Dump(value)

				err = fmt.Errorf("(NewArray) NewCWLType returned: %s", err.Error())
				return
			}

			array = append(array, value_cwl)
		}

		array_ptr = &array

	case []map[string]interface{}:
		native_array := native.([]map[string]interface{})

		array := Array{}
		for _, value := range native_array {

			var value_cwl CWLType
			value_cwl, err = NewCWLType("", value, context)
			if err != nil {
				fmt.Println("NewArray element:")
				spew.Dump(value)

				err = fmt.Errorf("(NewArray) NewCWLType returned: %s", err.Error())
				return
			}

			array = append(array, value_cwl)
		}

		array_ptr = &array

	default:
		err = fmt.Errorf("(NewArray) type not supported: %s", reflect.TypeOf(native))

	}

	return
}

func (c *Array) String() string {

	a_byte, err := json.Marshal(c)
	if err != nil {
		return err.Error()
	}

	return string(a_byte[:])
}

func (c *Array) Len() int {

	o := []CWLType(*c)

	return len(o)
}
