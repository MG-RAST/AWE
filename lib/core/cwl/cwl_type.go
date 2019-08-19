package cwl

import (
	"fmt"
	"math"
	"path"
	"reflect"
	"strings"

	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
	"gopkg.in/mgo.v2/bson"
)

// CWL Types
// http://www.commonwl.org/draft-3/CommandLineTool.html#CWLType
const (
	CWLNull      CWLType_Type_Basic = "null"      //no value
	CWLBoolean   CWLType_Type_Basic = "boolean"   //a binary value
	CWLInt       CWLType_Type_Basic = "int"       //32-bit signed integer
	CWLLong      CWLType_Type_Basic = "long"      //64-bit signed integer
	CWLFloat     CWLType_Type_Basic = "float"     //single precision (32-bit) IEEE 754 floating-point number
	CWLDouble    CWLType_Type_Basic = "double"    //double precision (64-bit) IEEE 754 floating-point number
	CWLString    CWLType_Type_Basic = "string"    //Unicode character sequence
	CWLFile      CWLType_Type_Basic = "File"      //A File object
	CWLDirectory CWLType_Type_Basic = "Directory" //A Directory object
	CWLAny       CWLType_Type_Basic = "Any"       //A Any object

	CWLStdin  CWLType_Type_Basic = "stdin"
	CWLStdout CWLType_Type_Basic = "stdout"
	CWLStderr CWLType_Type_Basic = "stderr"

	CWLArray   CWLType_Type_Basic = "array"   // unspecific type, only useful as a struct with items
	CWLRecord  CWLType_Type_Basic = "record"  // unspecific type
	CWLEnum    CWLType_Type_Basic = "enum"    // unspecific type
	CWLPointer CWLType_Type_Basic = "pointer" // unspecific type
)

// CWL Classes (note that type names also are class names)
const (
	CWLWorkflow          string = "Workflow"
	CWLCommandLineTool   string = "CommandLineTool"
	CWLWorkflowStepInput string = "WorkflowStepInput"
)

// ValidClasses _
var ValidClasses = []string{"Workflow", "CommandLineTool", "WorkflowStepInput"}

// ValidTypes _
var ValidTypes = []CWLType_Type_Basic{CWLNull, CWLBoolean, CWLInt, CWLLong, CWLFloat, CWLDouble, CWLString, CWLFile, CWLDirectory, CWLAny, CWLStdin, CWLStdout, CWLStderr}

// ValidClassMap _
var ValidClassMap = map[string]string{} // lower-case to correct case mapping

// ValidTypeMap _
var ValidTypeMap = map[CWLType_Type_Basic]CWLType_Type_Basic{} // lower-case to correct case mapping

// ArrayType _
type ArrayType interface {
	IsArrayType()
	Get_Array() *[]CWLType
}

//func (s CWLType_Type) Is_CommandOutputParameterType() {}

// CWLType - CWL basic types: int, string, boolean, .. etc
// http://www.commonwl.org/v1.0/CommandLineTool.html#CWLType
// null, boolean, int, long, float, double, string, File, Directory
type CWLType interface {
	CWLObject
	CWL_class // is an interface
	//Is_CommandInputParameterType()
	//Is_CommandOutputParameterType()
	GetType() CWLType_Type
	String() string
	//Is_Array() bool
	//IsCWLMinimal()
}

// CWLType_Impl _
type CWLType_Impl struct {
	CWLObjectImpl  `yaml:",inline" bson:",inline" json:",inline" mapstructure:",squash"` // provides: IsCWLObject()
	CWL_class_Impl `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // Provides: Id, Class
	Type           CWLType_Type                                                          `yaml:"-" json:"-" bson:"-" mapstructure:"-"`
}

// GetType _
func (c *CWLType_Impl) GetType() CWLType_Type { return c.Type }

// IsCWLMinimal _
func (c *CWLType_Impl) IsCWLMinimal() {}

// Is_CWLType _
func (c *CWLType_Impl) Is_CWLType() {}

//func (c *CWLType_Impl) Is_CommandInputParameterType()  {}
//func (c *CWLType_Impl) Is_CommandOutputParameterType() {}

func init() {

	for _, class := range ValidClasses {
		ValidClassMap[strings.ToLower(class)] = class
	}

	for _, a_type := range ValidTypes {
		a_type_str := string(a_type)
		lower := strings.ToLower(a_type_str)
		ValidTypeMap[CWLType_Type_Basic(lower)] = a_type
	}

}

// IsValidType _
func IsValidType(native_str string) (result CWLType_Type_Basic, ok bool) {
	native_str_lower := strings.ToLower(native_str)

	result, ok = ValidTypeMap[CWLType_Type_Basic(native_str_lower)]
	if !ok {
		return
	}

	return
}

// IsValidClass _
func IsValidClass(nativeStr string) (result string, ok bool) {
	nativeStrLower := strings.ToLower(nativeStr)

	result, ok = ValidClassMap[nativeStrLower]
	if !ok {
		return
	}

	return
}

//func (c *CWLType_Impl) Is_Array() bool                 { return false }

// NewCWLType _
func NewCWLType(id string, parentID string, native interface{}, context *WorkflowContext) (cwlType CWLType, err error) {

	//var cwl_type CWLType
	//fmt.Printf("(NewCWLType) starting with type %s\n", reflect.TypeOf(native))
	//fmt.Println("(NewCWLType)")
	//spew.Dump(native)

	native, err = MakeStringMap(native, context)
	if err != nil {
		return
	}

	//fmt.Printf("(NewCWLType) (B) starting with type %s\n", reflect.TypeOf(native))

	if native == nil {
		cwlType = NewNull()
		return
	}

	switch native.(type) {
	case []interface{}, []map[string]interface{}:
		//fmt.Printf("(NewCWLType) C\n")
		//nativeArray, _ := native.([]interface{})

		array, xerr := NewArray(id, parentID, native, context)
		if xerr != nil {
			err = fmt.Errorf("(NewCWLType) NewArray returned: %s", xerr.Error())
			return
		}
		cwlType = array

	case int:
		//fmt.Printf("(NewCWLType) D\n")
		nativeInt := native.(int)

		cwlType, err = NewInt(nativeInt, context)
		if err != nil {
			err = fmt.Errorf("(NewCWLType) NewInt: %s", err.Error())
			return
		}
	case int64:
		//fmt.Printf("(NewCWLType) D\n")
		nativeInt64 := native.(int64)

		cwlType = NewLong(nativeInt64)
	case float32:
		nativeFloat32 := native.(float32)
		cwlType = NewFloat(nativeFloat32)
	case float64:
		nativeFloat64 := native.(float64)
		cwlType = NewDouble(nativeFloat64)
	case string:
		//fmt.Printf("(NewCWLType) E\n")
		nativeStr := native.(string)

		cwlType = NewString(nativeStr)
	case bool:
		//fmt.Printf("(NewCWLType) F\n")
		nativeBool := native.(bool)

		cwlType = NewBoolean(id, nativeBool)

	case map[string]interface{}:
		//fmt.Printf("(NewCWLType) G\n")
		nativeMap := native.(map[string]interface{})

		_, hasItems := nativeMap["items"]

		if hasItems {
			cwlType, err = NewArray(id, parentID, nativeMap, context)
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

		cwlType, err = NewCWLTypeByClass(class, id, parentID, native, context)
		if err != nil {

			err = fmt.Errorf("(NewCWLType) NewCWLTypeByClass returned: %s", err.Error())
			return
		}

	case *File:
		cwlType = native.(*File)

	case *String:
		cwlType = native.(*String)
	case *Int:
		cwlType = native.(*Int)
	case *Boolean:
		cwlType = native.(*Boolean)

	default:
		//fmt.Printf("(NewCWLType) H\n")
		spew.Dump(native)
		err = fmt.Errorf("(NewCWLType) Type unknown: \"%s\" (%s)", reflect.TypeOf(native), spew.Sdump(native))
		return
	}
	//fmt.Printf("(NewCWLType) I\n")
	//if cwl_type.GetID() == "" {
	//	err = fmt.Errorf("(NewCWLType) Id is missing")
	//		return
	//	}

	//cwl_type_ptr = &cwl_type

	return

}

// NewCWLTypeByClass _
func NewCWLTypeByClass(class string, id string, parentID string, native interface{}, context *WorkflowContext) (cwlType CWLType, err error) {
	switch class {
	case string(CWLFile):
		//fmt.Println("NewCWLTypeByClass:")
		//spew.Dump(native)
		file, yerr := NewFileFromInterface(native, context, id)
		if yerr != nil {
			err = fmt.Errorf("(NewCWLTypeByClass) NewFile returned: %s", yerr.Error())
			return
		}
		cwlType = &file

	case string(CWLString):
		mystring, yerr := NewStringFromInterface(native)
		if yerr != nil {
			err = fmt.Errorf("(NewCWLTypeByClass) NewStringFromInterface returned: %s", yerr.Error())
			return
		}
		cwlType = mystring
	case string(CWLBoolean):
		myboolean, yerr := NewBooleanFromInterface(id, native)
		if yerr != nil {
			err = fmt.Errorf("(NewCWLTypeByClass) NewStringFromInterface returned: %s", yerr.Error())
			return
		}
		cwlType = myboolean
	case string(CWLDirectory):
		mydir, yerr := NewDirectoryFromInterface(native, context)
		if yerr != nil {
			err = fmt.Errorf("(NewCWLTypeByClass) NewDirectoryFromInterface returned: %s", yerr.Error())
			return
		}
		cwlType = mydir
	default:
		// Map type unknown, maybe a record
		//fmt.Println("This might be a record:")
		//spew.Dump(native)

		record, xerr := NewRecord(id, parentID, native, context)
		if xerr != nil {
			err = fmt.Errorf("(NewCWLTypeByClass) NewRecord returned: %s", xerr.Error())
			return
		}
		cwlType = &record
		return
	}
	return
}

func makeStringMapDeprecated(v interface{}) (result interface{}, err error) {

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

// NewCWLTypeArray _
func NewCWLTypeArray(native interface{}, parentID string, context *WorkflowContext) (cwlArrayPtr *[]CWLType, err error) {

	switch native.(type) {
	case []interface{}:

		nativeArray, ok := native.([]interface{})
		if !ok {
			err = fmt.Errorf("(NewCWLTypeArray) could not parse []interface{}")
			return
		}

		cwlArray := []CWLType{}

		for _, value := range nativeArray {
			valueCWL, xerr := NewCWLType("", "", value, context)
			if xerr != nil {
				err = xerr
				return
			}
			cwlArray = append(cwlArray, valueCWL)
		}
		cwlArrayPtr = &cwlArray
	default:

		ct, xerr := NewCWLType("", parentID, native, context)
		if xerr != nil {
			err = xerr
			return
		}

		cwlArrayPtr = &[]CWLType{ct}
	}

	return

}

// TypeIsCorrectSingle _
func TypeIsCorrectSingle(schema CWLType_Type, object CWLType, context *WorkflowContext) (ok bool, err error) {

	if schema == CWLAny {
		objectType := object.GetType()
		if objectType.Type2String() == string(CWLNull) {
			ok = false
			//err = fmt.Errorf("(TypeIsCorrectSingle) Any type does not accept Null")
			return
		}
		ok = true
		return

	}

	// resolve pointer
	pointer, isPointer := schema.(*Pointer)
	if isPointer {
		//resolve pointer / search schema

		_ = pointer
		fmt.Printf("schema needed: %s\n", string(*pointer))

		haveSchemaBase := path.Base(string(*pointer))

		for i := range context.Schemata {
			fmt.Printf("compare schema %s with %s\n", i, haveSchemaBase)
			if path.Base(i) == haveSchemaBase {
				ok = true
				return
			}
		}

	}

	switch object.(type) {
	case *Array:

		switch schema.(type) {
		case *CommandOutputArraySchema:
			ok = true
			return
		case *CommandInputArraySchema:
			ok = true
			return
		case OutputArraySchema:
			ok = true
			return
		case *OutputArraySchema:
			ok = true
			return
		case *ArraySchema:
			ok = true
			return
		case *InputArraySchema:
			ok = true
			return
		default:
			ok = false
			return
		}
	case *File:
		switch schema {
		case CWLStdout, CWLStderr, CWLFile:
			ok = true
			return
		default:
			ok = false
			return
		}

	default:

		objectType := object.GetType()
		objectTypeStr := objectType.Type2String()
		//fmt.Printf("TypeIsCorrectSingle: \"%s\" \"%s\"\n", reflect.TypeOf(schema), reflect.TypeOf(objectType))

		if schema.Type2String() == objectTypeStr {
			ok = true
			return
		}

		if objectTypeStr == "string" && schema.Type2String() == "EnumSchema" {

			var objectStr *String
			objectStr, ok = object.(*String)
			if !ok {
				err = fmt.Errorf("(TypeIsCorrectSingle) Could not convert to string")
				return
			}

			var enumSchema *EnumSchema

			switch schema.(type) {

			case *CommandInputEnumSchema:
				es := schema.(*CommandInputEnumSchema)
				enumSchema = &es.EnumSchema
			case *InputEnumSchema:
				es := schema.(*InputEnumSchema)
				enumSchema = &es.EnumSchema
			case *EnumSchema:
				enumSchema = schema.(*EnumSchema)

			default:
				err = fmt.Errorf("(TypeIsCorrectSingle) schema type unknown (%s)", reflect.TypeOf(schema))
				return
			}

			for _, symbol := range enumSchema.Symbols {
				symbolBase := path.Base(symbol)
				if symbolBase == objectStr.String() {
					ok = true // is valid enum element
					return
				}
			}

			spew.Dump(schema)
			panic("we are here")
		}

		// check if provided double can be excepted as int:
		if schema.Type2String() == string(CWLInt) && objectType.Type2String() == string(CWLDouble) {
			// now check if object is int anyway
			var d *Double
			d, ok = object.(*Double)
			if !ok {
				err = fmt.Errorf("(TypeIsCorrectSingle) Could not cast *Double")
				return
			}
			f := float64(*d)
			if f == math.Trunc(f) {
				ok = true
				return
			}

		}

		//fmt.Println("Comparsion:")
		//fmt.Printf("schema: %s\n", reflect.TypeOf(schema))
		//fmt.Printf("object: %s\n", reflect.TypeOf(object))
		//spew.Dump(schema)
		//spew.Dump(object)

		ok = false

	}

	return
}

// TypeIsCorrect _
func TypeIsCorrect(allowedTypes []CWLType_Type, object CWLType, context *WorkflowContext) (ok bool, err error) {

	for _, schema := range allowedTypes {

		var typeCorrect bool
		typeCorrect, err = TypeIsCorrectSingle(schema, object, context)
		if err != nil {
			return
		}
		if typeCorrect { //value == search_type {
			ok = true
			return
		} else {
			objectType := object.GetType()
			logger.Debug(3, "(HasInputParameterType) search_type %s does not macht expected type %s", objectType.Type2String(), schema.Type2String())

		}
	}

	ok = false

	return
}
