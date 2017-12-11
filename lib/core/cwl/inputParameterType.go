package cwl

import (
//"fmt"

//"github.com/davecgh/go-spew/spew"
//"reflect"
//"strings"
)

//type InputParameterType struct {
//	Type string
//}

type InputParameterType string

// func NewInputParameterType_dep(original interface{}) (ipt_ptr interface{}, err error) {
// 	fmt.Println("---- NewInputParameterType ----")
//
// 	//spew.Dump(original)
// 	//var ipt InputParameterType
//
// 	switch original.(type) {
// 	case []interface{}:
// 		original_array := original.([]interface{})
//
// 		array := []interface{}{}
//
// 		for _, v := range original_array {
// 			ipt, xerr := NewInputParameterType_dep(v)
// 			if xerr != nil {
// 				err = xerr
// 				return
// 			}
// 			array = append(array, ipt)
// 		}
// 		ipt_ptr = array
// 		return
//
// 	case map[string]interface{}:
// 		original_map := original.(map[string]interface{})
//
// 		type_value, has_type := original_map["type"]
// 		if !has_type {
// 			err = fmt.Errorf("(NewInputParameterType) type not found")
// 			return
// 		}
//
// 		if type_value == "array" {
// 			var array_schema *CommandOutputArraySchema
// 			array_schema, err = NewCommandOutputArraySchemaFromInterface(original_map)
// 			if err != nil {
// 				return
// 			}
// 			ipt_ptr = array_schema
// 			return
// 		} else {
// 			err = fmt.Errorf("(NewInputParameterType) type %s unknown", type_value)
// 			return
// 		}
//
// 	case string:
// 		original_str := original.(string)
//
// 		var original_type CWLType_Type
// 		original_type, err = NewCWLType_TypeFromString(original_str)
// 		if err != nil {
// 			err = fmt.Errorf("(NewInputParameterType) NewCWLType_TypeFromString returns %s", err.Error())
// 			return
// 		}
// 		ipt_ptr = original_type
// 		//ipt = InputParameterType(original_str)
//
// 		//ipt_ptr = &ipt
// 		return
// 	default:
// 		err = fmt.Errorf("(NewInputParameterType) type is not string: %s", reflect.TypeOf(original))
// 		return
// 	}
//
// 	return
// }

//func NewInputParameterTypeArray(original interface{}) (array_ptr *[]InputParameterType, err error) {
//	fmt.Println("---- NewInputParameterTypeArray ----")
// 	spew.Dump(original)
// 	array := []InputParameterType{}
//
// 	switch original.(type) {
//
// 	case string:
// 		ipt, xerr := NewInputParameterType(original)
// 		if xerr != nil {
// 			err = xerr
// 			return
// 		}
// 		array = append(array, *ipt)
// 		array_ptr = &array
// 		return
// 	default:
//
// 		spew.Dump(original)
// 		err = fmt.Errorf("(NewInputParameterTypeArray) type %s unknown", reflect.TypeOf(original))
// 		return
//
// 	}
// }
