package cwl

import (
//"fmt"
//"github.com/davecgh/go-spew/spew"
//"strings"
//"github.com/mitchellh/mapstructure"
//"reflect"
)

//type CommandInputParameterType struct {
//	Type string
//}

// CWLType | CommandInputRecordSchema | CommandInputEnumSchema | CommandInputArraySchema | string | array<CWLType | CommandInputRecordSchema | CommandInputEnumSchema | CommandInputArraySchema | string>

func NewCommandInputParameterTypeArray(original interface{}, schemata []CWLType_Type) (result []CWLType_Type, err error) {

	//result = []CWLType_Type{}
	return_array := []CWLType_Type{}

	switch original.(type) {
	case []interface{}:

		original_array := original.([]interface{})

		for _, element := range original_array {

			var cipt CWLType_Type
			cipt, err = NewCWLType_Type(schemata, element, "CommandInput")
			if err != nil {
				return
			}

			return_array = append(return_array, cipt)
		}

		result = return_array
		return

	default:

		var cipt CWLType_Type
		cipt, err = NewCWLType_Type(schemata, original, "CommandInput")
		if err != nil {
			return
		}
		return_array = append(return_array, cipt)
		result = return_array
		//err = fmt.Errorf("(NewCommandInputParameterTypeArray) Type %s unknown", reflect.TypeOf(original))
	}
	return
}

// func NewCommandInputParameterType(original interface{}) (result CWLType_Type, err error) {
//
// 	// Try CWL_Type
// 	//var cipt CommandInputParameterType
//
// 	original, err = MakeStringMap(original)
// 	if err != nil {
// 		return
// 	}
//
// 	switch original.(type) {
//
// 	case map[string]interface{}:
//
// 		original_map, ok := original.(map[string]interface{})
// 		if !ok {
// 			err = fmt.Errorf("(NewCommandInputParameterType) type error")
// 			return
// 		}
//
// 		type_str, ok := original_map["Type"]
// 		if !ok {
// 			type_str, ok = original_map["type"]
// 		}
//
// 		if !ok {
// 			err = fmt.Errorf("(NewCommandInputParameterType) type error, field type not found")
// 			return
// 		}
//
// 		switch type_str {
// 		case "array":
// 			schema, xerr := NewCommandInputArraySchemaFromInterface(original_map)
// 			if xerr != nil {
// 				err = xerr
// 				return
// 			}
// 			result = schema
// 			return
// 		case "enum":
//
// 			schema, xerr := NewCommandInputEnumSchema(original_map)
// 			if xerr != nil {
// 				err = xerr
// 				return
// 			}
// 			result = schema
// 			return
//
// 		case "record":
// 			schema, xerr := NewCommandInputRecordSchema(original_map)
// 			if xerr != nil {
// 				err = xerr
// 				return
// 			}
// 			result = schema
// 			return
//
// 		}
// 		err = fmt.Errorf("(NewCommandInputParameterType) type %s unknown", type_str)
// 		return
//
// 	case string:
// 		original_str := original.(string)
//
// 		result, err = NewCWLType_TypeFromString(original_str)
//
// 		// original_type := CWLType_Type_Basic(original_str)
// 		//
// 		// 		switch original_type {
// 		//
// 		// 		case CWL_null:
// 		// 		case CWL_boolean:
// 		// 		case CWL_int:
// 		// 		case CWL_long:
// 		// 		case CWL_float:
// 		// 		case CWL_double:
// 		// 		case CWL_string:
// 		// 		case CWL_File:
// 		// 		case CWL_Directory:
// 		// 		default:
// 		// 			err = fmt.Errorf("(NewCommandInputParameterType) type %s is unknown", original_str)
// 		// 			return
// 		// 		}
// 		// 		result = original_str
// 		return
// 	default:
// 		fmt.Printf("unknown type")
// 		spew.Dump(original)
// 		err = fmt.Errorf("(NewCommandInputParameterType) Type %s unknown", reflect.TypeOf(original))
// 		return
// 	}
// 	panic("do not come here")
// 	return
//
// }

//
// func HasCommandInputParameterType(array *[]CommandInputParameterType, search_type string) (ok bool) {
// 	for _, v := range *array {
// 		if v.Type == search_type {
// 			return true
// 		}
// 	}
// 	return false
// }
