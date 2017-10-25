package cwl

import (
	"fmt"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
	"gopkg.in/mgo.v2/bson"
	"reflect"
	"strings"
)

// CWL Types
// http://www.commonwl.org/draft-3/CommandLineTool.html#CWLType
const (
	CWL_null      CWLType_Type_Basic = "null"      //no value
	CWL_boolean   CWLType_Type_Basic = "boolean"   //a binary value
	CWL_int       CWLType_Type_Basic = "int"       //32-bit signed integer
	CWL_long      CWLType_Type_Basic = "long"      //64-bit signed integer
	CWL_float     CWLType_Type_Basic = "float"     //single precision (32-bit) IEEE 754 floating-point number
	CWL_double    CWLType_Type_Basic = "double"    //double precision (64-bit) IEEE 754 floating-point number
	CWL_string    CWLType_Type_Basic = "string"    //Unicode character sequence
	CWL_File      CWLType_Type_Basic = "File"      //A File object
	CWL_Directory CWLType_Type_Basic = "Directory" //A Directory object

	CWL_stdout CWLType_Type_Basic = "stdout"
	CWL_stderr CWLType_Type_Basic = "stderr"

	CWL_array  CWLType_Type_Basic = "array"  // unspecific type, only useful as a struct with items
	CWL_record CWLType_Type_Basic = "record" // unspecific type
	CWL_enum   CWLType_Type_Basic = "enum"   // unspecific type
)

// CWL Classes (note that type names also are class names)
const (
	CWL_Workflow          string = "Workflow"
	CWL_CommandLineTool   string = "CommandLineTool"
	CWL_WorkflowStepInput string = "WorkflowStepInput"
)

var Valid_Classes = []string{"Workflow", "CommandLineTool", "WorkflowStepInput"}

var Valid_Types = []CWLType_Type_Basic{CWL_null, CWL_boolean, CWL_int, CWL_long, CWL_float, CWL_double, CWL_string, CWL_File, CWL_Directory, CWL_stdout, CWL_stderr}

var ValidClassMap = map[string]string{}                        // lower-case to correct case mapping
var ValidTypeMap = map[CWLType_Type_Basic]CWLType_Type_Basic{} // lower-case to correct case mapping

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

//func (s CWLType_Type) Is_CommandOutputParameterType() {}

// CWLType - CWL basic types: int, string, boolean, .. etc
// http://www.commonwl.org/v1.0/CommandLineTool.html#CWLType
// null, boolean, int, long, float, double, string, File, Directory
type CWLType interface {
	CWL_object // is an interface
	//Is_CommandInputParameterType()
	//Is_CommandOutputParameterType()
	GetType() CWLType_Type
	String() string
	//Is_Array() bool
	//Is_CWL_minimal()
}

type CWLType_Impl struct {
	CWL_object_Impl `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // Provides: Id, Class
	Type            CWLType_Type                                                          `yaml:"-" json:"-" bson:"-" mapstructure:"-"`
}

func (c *CWLType_Impl) GetType() CWLType_Type { return c.Type }

func (c *CWLType_Impl) Is_CWL_minimal() {}
func (c *CWLType_Impl) Is_CWLType()     {}

//func (c *CWLType_Impl) Is_CommandInputParameterType()  {}
//func (c *CWLType_Impl) Is_CommandOutputParameterType() {}

func init() {

	for _, class := range Valid_Classes {
		ValidClassMap[strings.ToLower(class)] = class
	}

	for _, a_type := range Valid_Types {
		a_type_str := string(a_type)
		lower := strings.ToLower(a_type_str)
		ValidTypeMap[CWLType_Type_Basic(lower)] = a_type
	}

}

func IsValidType(native_str string) (result CWLType_Type_Basic, ok bool) {
	native_str_lower := strings.ToLower(native_str)

	result, ok = ValidTypeMap[CWLType_Type_Basic(native_str_lower)]
	if !ok {
		return
	}

	return
}

func IsValidClass(native_str string) (result string, ok bool) {
	native_str_lower := strings.ToLower(native_str)

	result, ok = ValidClassMap[native_str_lower]
	if !ok {
		return
	}

	return
}

//func (c *CWLType_Impl) Is_Array() bool                 { return false }

func NewCWLType(id string, native interface{}) (cwl_type CWLType, err error) {

	//var cwl_type CWLType
	fmt.Printf("(NewCWLType) starting with type %s\n", reflect.TypeOf(native))

	native, err = MakeStringMap(native)
	if err != nil {
		return
	}

	fmt.Printf("(NewCWLType) (B) starting with type %s\n", reflect.TypeOf(native))

	switch native.(type) {
	case []interface{}:
		fmt.Printf("(NewCWLType) C\n")
		native_array, _ := native.([]interface{})

		array, xerr := NewArray(id, native_array)
		if xerr != nil {
			err = fmt.Errorf("(NewCWLType) NewArray returned: ", xerr.Error())
			return
		}
		cwl_type = array

	case int:
		fmt.Printf("(NewCWLType) D\n")
		native_int := native.(int)

		cwl_type = NewInt(id, native_int)
	case float32:
		native_float32 := native.(float32)
		cwl_type = NewFloat(native_float32)
	case float64:
		native_float64 := native.(float64)
		cwl_type = NewDouble(native_float64)
	case string:
		fmt.Printf("(NewCWLType) E\n")
		native_str := native.(string)

		cwl_type = NewString(id, native_str)
	case bool:
		fmt.Printf("(NewCWLType) F\n")
		native_bool := native.(bool)

		cwl_type = NewBoolean(id, native_bool)

	case map[string]interface{}:
		fmt.Printf("(NewCWLType) G\n")
		native_map := native.(map[string]interface{})

		_, has_items := native_map["items"]

		if has_items {
			cwl_type, err = NewArray(id, native_map)
			if err != nil {
				err = fmt.Errorf("(NewCWLType) NewArray returned: %s", err.Error())
			}
			return

		}

		class, xerr := GetClass(native)
		if xerr != nil {

			err = fmt.Errorf("(NewCWLType) GetClass returned: %s", xerr.Error())
			return
		}

		cwl_type, err = NewCWLTypeByClass(class, id, native)
		if err != nil {

			err = fmt.Errorf("(NewCWLType) NewCWLTypeByClass returned: %s", err.Error())
			return
		}

	case *File:
		cwl_type = native.(*File)

	case *String:
		cwl_type = native.(*String)
	case *Int:
		cwl_type = native.(*Int)
	case *Boolean:
		cwl_type = native.(*Boolean)

	default:
		fmt.Printf("(NewCWLType) H\n")
		spew.Dump(native)
		err = fmt.Errorf("(NewCWLType) Type unknown: \"%s\" (%s)", reflect.TypeOf(native), spew.Sdump(native))
		return
	}
	fmt.Printf("(NewCWLType) I\n")
	//if cwl_type.GetId() == "" {
	//	err = fmt.Errorf("(NewCWLType) Id is missing")
	//		return
	//	}

	//cwl_type_ptr = &cwl_type

	return

}

func NewCWLTypeByClass(class string, id string, native interface{}) (cwl_type CWLType, err error) {
	switch class {
	case string(CWL_File):
		file, yerr := NewFile(id, native)
		cwl_type = &file
		if yerr != nil {
			err = fmt.Errorf("(NewCWLTypeByClass) NewFile returned: %s", yerr.Error())
			return
		}
	case string(CWL_string):
		mystring, yerr := NewStringFromInterface(id, native)
		if yerr != nil {
			err = fmt.Errorf("(NewCWLTypeByClass) NewStringFromInterface returned: %s", yerr.Error())
			return
		}
		cwl_type = mystring
	case string(CWL_boolean):
		myboolean, yerr := NewBooleanFromInterface(id, native)
		if yerr != nil {
			err = fmt.Errorf("(NewCWLTypeByClass) NewStringFromInterface returned: %s", yerr.Error())
			return
		}
		cwl_type = myboolean
	case string(CWL_Directory):
		mydir, yerr := NewDirectoryFromInterface(native)
		if yerr != nil {
			err = fmt.Errorf("(NewCWLTypeByClass) NewDirectoryFromInterface returned: %s", yerr.Error())
			return
		}
		cwl_type = mydir
	default:
		// Map type unknown, maybe a record
		fmt.Println("This might be a record:")
		spew.Dump(native)

		record, xerr := NewRecord(id, native)
		if xerr != nil {
			err = fmt.Errorf("(NewCWLTypeByClass) NewRecord returned: %s", xerr.Error())
			return
		}
		cwl_type = record
		return
	}
	return
}

func makeStringMap_deprecated(v interface{}) (result interface{}, err error) {

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

func NewCWLTypeArray(native interface{}, parent_id string) (cwl_array_ptr *[]CWLType, err error) {

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

func TypeIsCorrectSingle(schema CWLType_Type, object CWLType) (ok bool, err error) {

	switch object.(type) {
	case *Array:

		switch schema.(type) {
		case *CommandOutputArraySchema:
			ok = true
			return
		default:
			panic("array did not match")
		}
	default:

		object_type := object.GetType()

		if schema == object_type {
			ok = true
			return
		}

		fmt.Println("Comparsion:")
		fmt.Printf("schema: %s\n", reflect.TypeOf(schema))
		fmt.Printf("object: %s\n", reflect.TypeOf(object))
		spew.Dump(schema)
		spew.Dump(object)

		ok = false
		return

	}

	return
}

func TypeIsCorrect(allowed_types []CWLType_Type, object CWLType) (ok bool, err error) {

	for _, schema := range allowed_types {

		var type_correct bool
		type_correct, err = TypeIsCorrectSingle(schema, object)
		if err != nil {
			return
		}
		if type_correct { //value == search_type {
			ok = true
			return
		} else {
			object_type := object.GetType()
			logger.Debug(3, "(HasInputParameterType) search_type %s does not macht expected type %s", object_type.Type2String(), schema.Type2String())

		}
	}

	ok = false
	return

	return
}
