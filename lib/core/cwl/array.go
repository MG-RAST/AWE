package cwl

import (
	"fmt"
	"reflect"

	//"reflect"
	"encoding/json"

	"github.com/davecgh/go-spew/spew"
)

// Array _
type Array []CWLType

// IsCWLObject _
func (c *Array) IsCWLObject() {}

// GetClass _
func (c *Array) GetClass() string { return "array" }

// GetID _
func (c *Array) GetID() string { return "foobar" }

// SetID _
func (c *Array) SetID(string) {}

// GetType _
func (c *Array) GetType() CWLType_Type { return CWLArray }

// IsCWLMinimal _
func (c *Array) IsCWLMinimal() {}

//func (c *Array) Is_CommandInputParameterType()  {}
//func (c *Array) Is_CommandOutputParameterType() {}

// NewArray _
func NewArray(id string, parentID string, native interface{}, context *WorkflowContext) (arrayPtr *Array, err error) {

	switch native.(type) {
	case []interface{}:
		nativeArray := native.([]interface{})

		array := Array{}
		for _, value := range nativeArray {

			var valueCWL CWLType
			valueCWL, err = NewCWLType("", parentID, value, context)
			if err != nil {
				fmt.Println("NewArray element:")
				spew.Dump(value)

				err = fmt.Errorf("(NewArray) NewCWLType returned: %s", err.Error())
				return
			}

			array = append(array, valueCWL)
		}

		arrayPtr = &array

	case []map[string]interface{}:
		nativeArray := native.([]map[string]interface{})

		array := Array{}
		for _, value := range nativeArray {

			var valueCWL CWLType
			valueCWL, err = NewCWLType("", parentID, value, context)
			if err != nil {
				fmt.Println("NewArray element:")
				spew.Dump(value)

				err = fmt.Errorf("(NewArray) NewCWLType returned: %s", err.Error())
				return
			}

			array = append(array, valueCWL)
		}

		arrayPtr = &array

	default:
		err = fmt.Errorf("(NewArray) type not supported: %s", reflect.TypeOf(native))

	}

	return
}

// String _
func (c *Array) String() string {

	aByte, err := json.Marshal(c)
	if err != nil {
		return err.Error()
	}

	return string(aByte[:])
}

// Len _
func (c *Array) Len() int {

	o := []CWLType(*c)

	return len(o)
}
