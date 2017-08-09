package cwl

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"reflect"
)

// http://www.commonwl.org/draft-3/CommandLineTool.html#CWLType
const (
	CWL_null      = "null"      //no value
	CWL_boolean   = "boolean"   //a binary value
	CWL_int       = "int"       //32-bit signed integer
	CWL_long      = "long"      //64-bit signed integer
	CWL_float     = "float"     //single precision (32-bit) IEEE 754 floating-point number
	CWL_double    = "double"    //double precision (64-bit) IEEE 754 floating-point number
	CWL_string    = "string"    //Unicode character sequence
	CWL_File      = "File"      //A File object
	CWL_Directory = "Directory" //A Directory object

	CWL_array  = "array"
	CWL_record = "record"
	CWL_enum   = "enum"

	CWL_stdout = "stdout"
	CWL_stderr = "stderr"

	CWL_boolean_array   = "boolean_array"   //a binary value
	CWL_int_array       = "int_array"       //32-bit signed integer
	CWL_long_array      = "long_array"      //64-bit signed integer
	CWL_float_array     = "float_array"     //single precision (32-bit) IEEE 754 floating-point number
	CWL_double_array    = "double_array"    //double precision (64-bit) IEEE 754 floating-point number
	CWL_string_array    = "string_array"    //Unicode character sequence
	CWL_File_array      = "File_array"      //A File object
	CWL_Directory_array = "Directory_array" //A Directory object
)

var Valid_cwltypes = map[string]bool{
	CWL_null:            true,
	CWL_boolean:         true,
	CWL_int:             true,
	CWL_long:            true,
	CWL_float:           true,
	CWL_double:          true,
	CWL_string:          true,
	CWL_File:            true,
	CWL_Directory:       true,
	CWL_stdout:          true,
	CWL_stderr:          true,
	CWL_boolean_array:   true,
	CWL_int_array:       true,
	CWL_long_array:      true,
	CWL_float_array:     true,
	CWL_double_array:    true,
	CWL_string_array:    true,
	CWL_File_array:      true,
	CWL_Directory_array: true,
}

var Native2Array = map[string]string{
	CWL_boolean:   CWL_boolean_array,
	CWL_int:       CWL_int_array,
	CWL_long:      CWL_long_array,
	CWL_float:     CWL_float_array,
	CWL_double:    CWL_double_array,
	CWL_string:    CWL_string_array,
	CWL_File:      CWL_File_array,
	CWL_Directory: CWL_Directory_array,
}

type CWL_array_type interface {
	Is_CWL_array_type()
	Get_Array() *[]CWLType
}

type CWL_minimal_interface interface {
	Is_CWL_minimal()
}

type CWL_minimal struct{}

func (c *CWL_minimal) Is_CWL_minimal() {}

// generic class to represent Files and Directories
type CWL_location interface {
	GetLocation() string
}

// CWLType - CWL basic types: int, string, boolean, .. etc
// http://www.commonwl.org/v1.0/CommandLineTool.html#CWLType
// null, boolean, int, long, float, double, string, File, Directory
type CWLType interface {
	CWL_object // is an interface
	Is_CommandInputParameterType()
	Is_CommandOutputParameterType()
	Is_CWLType()
	String() string
	//Is_Array() bool
	//Is_CWL_minimal()
}

type CWLType_Impl struct{}

func (c *CWLType_Impl) Is_CWL_minimal()                {}
func (c *CWLType_Impl) Is_CWLType()                    {}
func (c *CWLType_Impl) Is_CommandInputParameterType()  {}
func (c *CWLType_Impl) Is_CommandOutputParameterType() {}

//func (c *CWLType_Impl) Is_Array() bool                 { return false }

func NewCWLType(id string, native interface{}) (cwl_type CWLType, err error) {

	//var cwl_type CWLType

	switch native.(type) {
	case []interface{}:

		native_array, _ := native.([]interface{})

		array, xerr := NewArray(id, native_array)
		if xerr != nil {
			err = xerr
			return
		}
		cwl_type = array
		return

	case int:
		native_int := native.(int)

		cwl_type = &Int{Value: native_int}

		//cwl_type = int_type.(*CWLType)

		return
		//temp = &Int{Value: native_int}

	case string:
		native_str := native.(string)

		cwl_type = NewString("", native_str)
	case bool:
		native_bool := native.(bool)

		cwl_type = &Boolean{Value: native_bool}

	case map[interface{}]interface{}:

		class, xerr := GetClass(native)
		if xerr != nil {
			err = xerr
			return
		}
		cwl_type, err = NewCWLTypeByClass(class, native)
		return

	case map[string]interface{}:
		//empty, xerr := NewEmpty(native)
		//if xerr != nil {
		//	err = xerr
		//	return
		//}
		class, xerr := GetClass(native)
		if xerr != nil {
			err = xerr
			return
		}
		cwl_type, err = NewCWLTypeByClass(class, native)
		return

	default:
		spew.Dump(native)
		err = fmt.Errorf("(NewCWLType) Type unknown: \"%s\"", reflect.TypeOf(native))
		return
	}
	//cwl_type_ptr = &cwl_type

	return

}

func NewCWLTypeByClass(class string, native interface{}) (cwl_type CWLType, err error) {
	switch class {
	case CWL_File:
		file, yerr := NewFile(native)
		cwl_type = &file
		if yerr != nil {
			err = yerr
			return
		}
	case CWL_string:
		mystring, yerr := NewStringFromInterface(native)
		cwl_type = mystring
		if yerr != nil {
			err = yerr
			return
		}
	default:
		// Map type unknown, maybe a record
		spew.Dump(native)

		record, xerr := NewRecord(native)
		if xerr != nil {
			err = xerr
			return
		}
		cwl_type = record
		return
	}
	return
}

func NewCWLTypeArray_deprecated(native interface{}) (cwl_array_ptr *[]CWLType, err error) {

	switch native.(type) {
	case []interface{}:

		native_array, ok := native.([]interface{})
		if !ok {
			err = fmt.Errorf("(NewCWLTypeArray) could not parse []interface{}")
			return
		}

		cwl_array := []CWLType{}

		for _, value := range native_array {
			value_cwl, xerr := NewCWLType("", value)
			if xerr != nil {
				err = xerr
				return
			}
			cwl_array = append(cwl_array, value_cwl)
		}
		cwl_array_ptr = &cwl_array
	default:

		ct, xerr := NewCWLType("", native)
		if xerr != nil {
			err = xerr
			return
		}

		cwl_array_ptr = &[]CWLType{ct}
	}

	return

}
