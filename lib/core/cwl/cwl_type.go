package cwl

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"gopkg.in/mgo.v2/bson"
	"reflect"
	"strings"
)

// http://www.commonwl.org/draft-3/CommandLineTool.html#CWLType
const (
	CWL_null      CWLType_Type = "null"      //no value
	CWL_boolean   CWLType_Type = "boolean"   //a binary value
	CWL_int       CWLType_Type = "int"       //32-bit signed integer
	CWL_long      CWLType_Type = "long"      //64-bit signed integer
	CWL_float     CWLType_Type = "float"     //single precision (32-bit) IEEE 754 floating-point number
	CWL_double    CWLType_Type = "double"    //double precision (64-bit) IEEE 754 floating-point number
	CWL_string    CWLType_Type = "string"    //Unicode character sequence
	CWL_File      CWLType_Type = "File"      //A File object
	CWL_Directory CWLType_Type = "Directory" //A Directory object

	CWL_array  CWLType_Type = "array"  // unspecific type
	CWL_record CWLType_Type = "record" // unspecific type
	CWL_enum   CWLType_Type = "enum"   // unspecific type

	CWL_stdout CWLType_Type = "stdout"
	CWL_stderr CWLType_Type = "stderr"

	CWL_boolean_array   CWLType_Type = "boolean_array"   //a binary value
	CWL_int_array       CWLType_Type = "int_array"       //32-bit signed integer
	CWL_long_array      CWLType_Type = "long_array"      //64-bit signed integer
	CWL_float_array     CWLType_Type = "float_array"     //single precision (32-bit) IEEE 754 floating-point number
	CWL_double_array    CWLType_Type = "double_array"    //double precision (64-bit) IEEE 754 floating-point number
	CWL_string_array    CWLType_Type = "string_array"    //Unicode character sequence
	CWL_File_array      CWLType_Type = "File_array"      //A File object
	CWL_Directory_array CWLType_Type = "Directory_array" //A Directory object

	CWL_Workflow          CWLType_Type = "Workflow"
	CWL_CommandLineTool   CWLType_Type = "CommandLineTool"
	CWL_WorkflowStepInput CWLType_Type = "WorkflowStepInput"
)

var Valid_cwltypes = map[CWLType_Type]bool{
	CWL_null:      true,
	CWL_boolean:   true,
	CWL_int:       true,
	CWL_long:      true,
	CWL_float:     true,
	CWL_double:    true,
	CWL_string:    true,
	CWL_File:      true,
	CWL_Directory: true,
	CWL_stdout:    true,
	CWL_stderr:    true,
	//CWL_boolean_array:   true,
	//CWL_int_array:       true,
	//CWL_long_array:      true,
	//CWL_float_array:     true,
	//CWL_double_array:    true,
	//CWL_string_array:    true,
	//CWL_File_array:      true,
	//CWL_Directory_array: true,
}

var Valid_class_types = map[CWLType_Type]bool{
	CWL_Workflow:          true,
	CWL_CommandLineTool:   true,
	CWL_WorkflowStepInput: true,
}

var Lowercase_fix = map[string]CWLType_Type{
	"file":              CWL_File,
	"directory":         CWL_Directory,
	"workflow":          CWL_Workflow,
	"commandlinetool":   CWL_CommandLineTool,
	"workflowstepinput": CWL_WorkflowStepInput,
}

var Native2Array = map[CWLType_Type]CWLType_Type{
	CWL_boolean:   CWL_boolean_array,
	CWL_int:       CWL_int_array,
	CWL_long:      CWL_long_array,
	CWL_float:     CWL_float_array,
	CWL_double:    CWL_double_array,
	CWL_string:    CWL_string_array,
	CWL_File:      CWL_File_array,
	CWL_Directory: CWL_Directory_array,
}

func NewCWLType_TypeFromString(native string) (result CWLType_Type, err error) {
	native_str_lower := strings.ToLower(native)

	var ok bool
	result, ok = Lowercase_fix[native_str_lower]
	if ok {
		return
	}

	other := CWLType_Type(native_str_lower)

	_, ok = Valid_cwltypes[other]
	if !ok {

		_, ok = Valid_class_types[other]
		if !ok {

			err = fmt.Errorf("(NewCWLType_TypeFromString) type %s unkown", native_str_lower)
			return
		}

	}
	result = other

	return
}

func NewCWLType_Type(native interface{}) (result CWLType_Type, err error) {
	switch native.(type) {
	case string:
		native_str := native.(string)

		return NewCWLType_TypeFromString(native_str)

	default:
		err = fmt.Errorf("(NewCWLType_Type) type %s unkown", reflect.TypeOf(native))
		return
	}
}

func NewCWLType_TypeArray(native interface{}) (result []CWLType_Type, err error) {

	switch native.(type) {
	case map[string]interface{}:

		original_map := native.(map[string]interface{})

		type_value, has_type := original_map["type"]
		if !has_type {
			err = fmt.Errorf("(NewCWLType_TypeArray) type not found")
			return
		}

		if type_value == "array" {
			var array_schema *CommandOutputArraySchema
			array_schema, err = NewCommandOutputArraySchema(original_map)
			if err != nil {
				return
			}
			result = []CWLType_Type{array_schema}
			return
		} else {
			err = fmt.Errorf("(NewCWLType_TypeArray) type %s unknown", type_value)
			return
		}

	case string:
		native_str := native.(string)
		var a_type CWLType_Type
		a_type, err = NewCWLType_TypeFromString(native_str)
		if err != nil {
			return
		}
		result = []CWLType_Type{a_type}

	case []string:

		native_array := native.([]string)
		type_array := []CWLType_Type{}
		for _, element_str := range native_array {
			var element_type CWLType_Type
			element_type, err = NewCWLType_TypeFromString(element_str)
			if err != nil {
				return
			}
			type_array = append(type_array, element_type)
		}

		result = type_array

	default:
		spew.Dump(native)
		err = fmt.Errorf("(NewCWLType_TypeArray) type unknown: %s", reflect.TypeOf(native))

	}
	return
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

type CWLType_Type string

//func (s CWLType_Type) Is_CommandOutputParameterType() {}
//func (s CWLType_Type) Is_CWLType_Type() {}

const mytest CWLType_Type = "hello"

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

type CWLType_Impl struct {
	Id string `yaml:"id,omitempty" json:"id,omitempty" bson:"id,omitempty"`
}

func (c *CWLType_Impl) GetId() string   { return c.Id }
func (c *CWLType_Impl) SetId(id string) { c.Id = id }

func (c *CWLType_Impl) Is_CWL_minimal()                {}
func (c *CWLType_Impl) Is_CWLType()                    {}
func (c *CWLType_Impl) Is_CommandInputParameterType()  {}
func (c *CWLType_Impl) Is_CommandOutputParameterType() {}

//func (c *CWLType_Impl) Is_Array() bool                 { return false }

func NewCWLType(id string, native interface{}) (cwl_type CWLType, err error) {

	//var cwl_type CWLType

	native, err = makeStringMap(native)
	if err != nil {
		return
	}

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

		cwl_type = NewInt(id, native_int)

		//cwl_type = int_type.(*CWLType)

		return
		//temp = &Int{Value: native_int}

	case string:
		native_str := native.(string)

		cwl_type = NewString(id, native_str)
	case bool:
		native_bool := native.(bool)

		cwl_type = NewBoolean(id, native_bool)

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
		cwl_type, err = NewCWLTypeByClass(class, id, native)
		return
	case map[interface{}]interface{}:
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
		cwl_type, err = NewCWLTypeByClass(class, id, native)
		return
	default:
		spew.Dump(native)
		err = fmt.Errorf("(NewCWLType) Type unknown: \"%s\"", reflect.TypeOf(native))
		return
	}
	//cwl_type_ptr = &cwl_type

	return

}

func NewCWLTypeByClass(class CWLType_Type, id string, native interface{}) (cwl_type CWLType, err error) {
	switch class {
	case CWL_File:
		file, yerr := NewFile(id, native)
		cwl_type = &file
		if yerr != nil {
			err = yerr
			return
		}
	case CWL_string:
		mystring, yerr := NewStringFromInterface(id, native)
		cwl_type = mystring
		if yerr != nil {
			err = yerr
			return
		}
	default:
		// Map type unknown, maybe a record
		spew.Dump(native)

		record, xerr := NewRecord(id, native)
		if xerr != nil {
			err = xerr
			return
		}
		cwl_type = record
		return
	}
	return
}

func makeStringMap(v interface{}) (result interface{}, err error) {

	switch v.(type) {
	case bson.M:

		original_map := v.(bson.M)

		new_map := make(map[string]interface{})

		for key_str, value := range original_map {

			new_map[key_str] = value
		}

		result = new_map
		return
	case map[interface{}]interface{}:

		v_map, ok := v.(map[interface{}]interface{})
		if !ok {
			err = fmt.Errorf("casting problem (b)")
			return
		}
		v_string_map := make(map[string]interface{})

		for key, value := range v_map {
			key_string := key.(string)
			v_string_map[key_string] = value
		}

		result = v_string_map
		return

	}
	result = v
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
