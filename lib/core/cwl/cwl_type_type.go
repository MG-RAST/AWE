package cwl

import (
	"fmt"
	"path"
	"reflect"
	"strings"

	"github.com/davecgh/go-spew/spew"
)

type CWLType_Type interface {
	Is_Type()
	Type2String() string
	GetID() string
}

type CWLType_Type_Basic string

// Is_Type _
func (s CWLType_Type_Basic) Is_Type() {}

// Type2String _
func (s CWLType_Type_Basic) Type2String() string { return string(s) }

// GetID _
func (s CWLType_Type_Basic) GetID() string { return "" }

// NewCWLType_TypeFromString _
func NewCWLType_TypeFromString(schemata []CWLType_Type, native string, contextStr string, context *WorkflowContext) (result []CWLType_Type, err error) {

	if native == "" {
		err = fmt.Errorf("(NewCWLType_TypeFromString) string is empty")
		return
	}

	result = []CWLType_Type{}

	// if strings.HasPrefix(native, "#") {
	// 	result = append(result, NewPointerFromstring(native))

	// 	return
	// }

	//fmt.Printf("(NewCWLType_TypeFromString) native: %s\n", native)

	if strings.Contains(native, "#") {
		//fmt.Printf("(NewCWLType_TypeFromString) contains\n")
		result = append(result, NewPointerFromstring(native))
		return
	} //else {
	//fmt.Printf("(NewCWLType_TypeFromString) does not contain\n")
	//}

	//fmt.Printf("(NewCWLType_TypeFromString) native: %s\n", native)

	if strings.HasSuffix(native, "?") { // optional argument
		native = strings.TrimSuffix(native, "?")
		result = append(result, CWLNull)
	}

	if !strings.HasSuffix(native, "[]") {

		var ok bool
		var basicResult CWLType_Type
		basicResult, ok = IsValidType(native)

		if !ok {
			//fmt.Printf("(NewCWLType_TypeFromString) native: %s not valid\n", native)
			//	var someSchemaDef CWLType_Type

			for name := range context.Schemata {
				if path.Base(native) == path.Base(name) {
					//fmt.Printf("(NewCWLType_TypeFromString) native: %s match\n", native)
					ok = true
					result = append(result, NewPointerFromstring(native))
					return
				}
			}
			//fmt.Printf("(NewCWLType_TypeFromString) native: %s no match\n", native)
			// _, ok = context.Schemata[native]
			// if ok {
			// 	// found Schema, add pointer NewPointerFromstring(native)
			// 	result = append(result, NewPointerFromstring(native))
			// 	return
			// }

			sList := ""
			for k := range context.Schemata {
				sList += "," + k
			}

			err = fmt.Errorf("(NewCWLType_TypeFromString) type %s unkown (found custom schemata: %s)", native, sList)
			return
		}

		result = append(result, basicResult)

		return
	}

	// is array

	//an_array_schema := make(map[string]interface{})
	//an_array_schema["type"] = "array"

	baseTypeStr := strings.TrimSuffix(native, "[]")
	var ok bool
	_, ok = IsValidType(baseTypeStr)

	if !ok {
		err = fmt.Errorf("(NewCWLType_TypeFromString) base_type %s unkown", baseTypeStr)
		return
	}

	// recurse:
	var baseTypeArray []CWLType_Type
	baseTypeArray, err = NewCWLType_TypeFromString(schemata, baseTypeStr, contextStr, context)
	if err != nil {
		err = fmt.Errorf("(NewCWLType_TypeFromString) recurisve call of NewCWLType_TypeFromString returned: %s", err.Error())
		return
	}

	arraySchema := ArraySchema{}
	arraySchema.Items = baseTypeArray
	switch contextStr {

	case "Output", "ExpressionToolOutput":

		oas := NewOutputArraySchema()
		oas.Items = baseTypeArray
		result = []CWLType_Type{oas}
		return
	case "CommandInput":

		cias := NewCommandInputArraySchema()
		cias.ArraySchema = arraySchema
		//cias.ArraySchema.Items = []CWLType_Type{baseType}
		result = []CWLType_Type{cias}
		return

	case "CommandOutput":

		cias := NewCommandOutputArraySchema()
		cias.ArraySchema = arraySchema

		result = []CWLType_Type{cias}
		return

	case "Input":
		ias := NewInputArraySchema()
		ias.ArraySchema = arraySchema
		result = []CWLType_Type{ias}
		return

	default:
		err = fmt.Errorf("(NewCWLType_TypeFromString) context %s not supported yet (baseTypeStr: %s, native: %s)", contextStr, baseTypeStr, native)
	}

	return
}

// NewCWLType_Type _
func NewCWLType_Type(schemata []CWLType_Type, native interface{}, context_p string, context *WorkflowContext) (result []CWLType_Type, err error) {

	native, err = MakeStringMap(native, context)
	if err != nil {
		return
	}

	switch native.(type) {
	case string:
		native_str := native.(string)

		result, err = NewCWLType_TypeFromString(schemata, native_str, context_p, context)
		if err != nil {
			err = fmt.Errorf("(NewCWLType_Type) A NewCWLType_TypeFromString returned: %s", err.Error())
		}

		return

	case CWLType_Type_Basic:

		native_tt := native.(CWLType_Type_Basic)
		native_str := string(native_tt)

		result, err = NewCWLType_TypeFromString(schemata, native_str, context_p, context)
		if err != nil {
			err = fmt.Errorf("(NewCWLType_Type) B NewCWLType_TypeFromString returned: %s", err.Error())
		}
		return

	case map[string]interface{}:
		nativeMap := native.(map[string]interface{})

		var objectTypeIf interface{}
		objectType := ""
		var hasType bool

		objectTypeIf, hasType = nativeMap["type"]

		if hasType {
			var ok bool
			objectType, ok = objectTypeIf.(string)
			if !ok {
				err = fmt.Errorf("(NewCWLType_Type) field type is not a string %s", err.Error())
				return
			}
		} else {
			//fmt.Println("(NewCWLType_Type) nativeMap:")
			//spew.Dump(nativeMap)

			//err = fmt.Errorf("(NewCWLType_Type) map object has not field \"type\" (context: %s)", context_p)
			//return

			_, hasItems := nativeMap["items"]
			if hasItems {
				objectType = "array"
			}

		}

		var identifierIf interface{}
		var hasID bool
		identifierIf, hasID = nativeMap["id"]
		if hasID {
			identifierStr := identifierIf.(string)

			identifierStrArray := strings.Split(identifierStr, "#")
			if len(identifierStrArray) > 1 {
				identifierStr = identifierStrArray[1]
				nativeMap["id"] = identifierStr
			}
		}

		switch objectType {
		case "":
			err = fmt.Errorf("(NewCWLType_Type) object_type could not be detected (context: %s)", context_p)
			return
		case "array":

			switch context_p {
			case "Input":
				var schema *InputArraySchema
				schema, err = NewInputArraySchemaFromInterface(native, schemata, context)
				if err != nil {
					err = fmt.Errorf("(NewCWLType_Type) NewInputArraySchemaFromInterface returned: %s", err.Error())
				}
				result = append(result, schema)
				return
			case "Output", "ExpressionToolOutput":
				var schema *OutputArraySchema
				schema, err = NewOutputArraySchemaFromInterface(native, schemata, context)
				if err != nil {
					err = fmt.Errorf("(NewCWLType_Type) NewOutputArraySchemaFromInterface returned: %s", err.Error())
				}
				result = append(result, schema)
				return
			case "CommandOutput": // CommandOutputRecordSchema | CommandOutputEnumSchema | CommandOutputArraySchema
				var schema *CommandOutputArraySchema
				schema, err = NewCommandOutputArraySchemaFromInterface(native, schemata, context)
				if err != nil {
					err = fmt.Errorf("(NewCWLType_Type) NewCommandOutputArraySchemaFromInterface returned: %s", err.Error())
				}
				result = append(result, schema)
				return
			case "CommandInput":
				var schema *CommandInputArraySchema
				schema, err = NewCommandInputArraySchemaFromInterface(native, schemata, context)
				if err != nil {
					fmt.Println("NewCWLType_Type calls NewCommandInputArraySchemaFromInterface")
					spew.Dump(native)
					err = fmt.Errorf("(NewCWLType_Type) NewCommandInputArraySchemaFromInterface returned: %s", err.Error())
				}
				result = append(result, schema)
				return
			default:
				err = fmt.Errorf("(NewCWLType_Type) array, context_p %s unknown", context_p)
				return
			}

		case "record":

			switch context_p {
			case "Input": // InputRecordSchema
				var schema *InputRecordSchema
				schema, err = NewInputRecordSchemaFromInterface(native, schemata, context)
				if err != nil {
					err = fmt.Errorf("(NewCWLType_Type) NewInputRecordSchemaFromInterface returned: %s", err.Error())
				}
				result = append(result, schema)
				return
			case "CommandInput":
				var schema *CommandInputRecordSchema
				schema, err = NewCommandInputRecordSchemaFromInterface(native, schemata, context)
				if err != nil {
					//spew.Dump(native)
					//panic("nooooo!")
					err = fmt.Errorf("(NewCWLType_Type) NewCommandInputRecordSchemaFromInterface returned: %s", err.Error())
				}
				result = append(result, schema)
				return
			case "CommandOutput":
				var schema *CommandOutputRecordSchema
				schema, err = NewCommandOutputRecordSchemaFromInterface(native, schemata, context)
				if err != nil {
					err = fmt.Errorf("(NewCWLType_Type) NewCommandOutputRecordSchemaFromInterface returned: %s", err.Error())
				}
				result = append(result, schema)
				return

			case "Output", "ExpressionToolOutput":
				var schema *OutputRecordSchema
				schema, err = NewOutputRecordSchemaFromInterface(native, schemata, context)
				if err != nil {
					err = fmt.Errorf("(NewCWLType_Type) NewOutputRecordSchemaFromInterface returned: %s", err.Error())
				}
				result = append(result, schema)
				return

			default:
				err = fmt.Errorf("(NewCWLType_Type) record, context %s unknown", context_p)
				return
			}

		case "enum":

			switch context_p {
			case "Input":
				var schema *InputEnumSchema
				schema, err = NewInputEnumSchemaFromInterface(native, context)
				if err != nil {
					err = fmt.Errorf("(NewCWLType_Type) NewInputEnumSchemaFromInterface returned: %s", err.Error())
				}
				result = append(result, schema)
				return
			case "Output", "ExpressionToolOutput":
				var schema *OutputEnumSchema
				schema, err = NewOutputEnumSchemaFromInterface(native, context)
				if err != nil {
					err = fmt.Errorf("(NewCWLType_Type) NewInputEnumSchemaFromInterface returned: %s", err.Error())
				}
				result = append(result, schema)
				return
			case "CommandInput":
				var schema *CommandInputEnumSchema
				schema, err = NewCommandInputEnumSchemaFromInterface(native, context)
				if err != nil {
					err = fmt.Errorf("(NewCWLType_Type) NewCommandInputEnumSchemaFromInterface returned: %s", err.Error())
				}
				result = append(result, schema)
				return
			default:
				err = fmt.Errorf("(NewCWLType_Type) enum, context_p %s unknown", context_p)
				return
			}
		default:

			fmt.Println("(NewCWLType_Type) nativeMap:")
			spew.Dump(nativeMap)
			err = fmt.Errorf("(NewCWLType_Type) object_type %s not supported yet", objectType)
			return
		}
	case OutputArraySchema:
		schema := native.(OutputArraySchema)
		result = append(result, schema)
		return
	case *OutputArraySchema:
		oasP := native.(*OutputArraySchema)
		result = append(result, *oasP)
		//result = *oas_p
	case *OutputRecordSchema:
		orsP := native.(*OutputRecordSchema)
		result = append(result, orsP)
	case *CommandOutputArraySchema:
		coasP := native.(*CommandOutputArraySchema)

		result = append(result, coasP)
	default:
		spew.Dump(native)
		err = fmt.Errorf("(NewCWLType_Type) type %s unkown", reflect.TypeOf(native))
		return
	}

	return
}

// NewCWLType_TypeArray _
func NewCWLType_TypeArray(native interface{}, schemata []CWLType_Type, contextP string, passSchemataAlong bool, context *WorkflowContext) (result []CWLType_Type, err error) {

	native, err = MakeStringMap(native, context)
	if err != nil {
		return
	}

	switch native.(type) {
	case map[string]interface{}:

		originalMap := native.(map[string]interface{})

		//var newTypeArray []CWLType_Type
		result, err = NewCWLType_Type(schemata, originalMap, contextP, context)

		if err != nil {
			err = fmt.Errorf("(NewCWLType_TypeArray) NewCWLType_Type returns: %s", err.Error())
			return
		}

		//result = newTypeArray
		return

	case string:
		nativeStr := native.(string)

		//var a_type CWLType_Type
		result, err = NewCWLType_TypeFromString(schemata, nativeStr, contextP, context)
		if err != nil {
			return
		}
		//result = []CWLType_Type{a_type}

	case []string:

		nativeArray := native.([]string)
		//type_array := []CWLType_Type{}
		for _, elementStr := range nativeArray {
			var elementStrArray []CWLType_Type
			elementStrArray, err = NewCWLType_TypeFromString(schemata, elementStr, contextP, context)
			if err != nil {
				return
			}
			for _, element := range elementStrArray {
				result = append(result, element)
			}

		}

		//result = type_array
		return

	case []interface{}:

		nativeArray := native.([]interface{})
		//type_array := []CWLType_Type{}
		for _, elementStr := range nativeArray {
			//var element_type CWLType_Type
			var elementStrArray []CWLType_Type
			elementStrArray, err = NewCWLType_Type(schemata, elementStr, contextP, context)
			if err != nil {
				return
			}
			for _, element := range elementStrArray {
				result = append(result, element)
				if passSchemataAlong {
					schemata = append(schemata, element)
				}
			}
			//type_array = append(type_array, element_type)

		}

		//result = type_array
		return

	case []CWLType_Type:
		result = native.([]CWLType_Type)
	default:
		spew.Dump(native)
		err = fmt.Errorf("(NewCWLType_TypeArray) type unknown: %s", reflect.TypeOf(native))

	}
	return
}
