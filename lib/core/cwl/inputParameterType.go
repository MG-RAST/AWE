package cwl

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"reflect"
	//"strings"
)

//type InputParameterType struct {
//	Type string
//}

type InputParameterType string

func NewInputParameterType(original interface{}) (ipt_ptr interface{}, err error) {
	fmt.Println("---- NewInputParameterType ----")

	spew.Dump(original)
	//var ipt InputParameterType

	switch original.(type) {
	case []interface{}:
		original_array := original.([]interface{})

		array := []interface{}{}

		for _, v := range original_array {
			ipt, xerr := NewInputParameterType(v)
			if xerr != nil {
				err = xerr
				return
			}
			array = append(array, ipt)
		}
		ipt_ptr = array
		return

	case map[string]interface{}:
		original_map := original.(map[string]interface{})

		type_value, has_type := original_map["type"]
		if !has_type {
			err = fmt.Errorf("(NewInputParameterType) type not found")
			return
		}

		if type_value == "array" {
			var array_schema *CommandOutputArraySchema
			array_schema, err = NewCommandOutputArraySchema(original_map)
			if err != nil {
				return
			}
			ipt_ptr = array_schema
			return
		} else {
			err = fmt.Errorf("(NewInputParameterType) type %s unknown", type_value)
			return
		}

	case string:
		original_str := original.(string)

		var original_type CWLType_Type
		original_type, err = NewCWLType_TypeFromString(original_str)
		if err != nil {
			err = fmt.Errorf("(NewInputParameterType) NewCWLType_TypeFromString returns %s", err.Error())
			return
		}
		ipt_ptr = original_type
		//ipt = InputParameterType(original_str)

		//ipt_ptr = &ipt
		return
	default:
		err = fmt.Errorf("(NewInputParameterType) type is not string: %s", reflect.TypeOf(original))
		return
	}

	return
}

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

func HasInputParameterType(allowed_types []CWLType_Type, search_type CWLType_Type) (ok bool, err error) {

	for _, value := range allowed_types {
		if value == search_type {
			ok = true
			return
		}
	}

	ok = false
	return

	// var array_of []interface{}
	//
	// switch array.(type) {
	// case *InputParameterType:
	//
	// 	ipt, tok := array.(*InputParameterType)
	// 	if !tok {
	// 		err = fmt.Errorf("(HasInputParameterType) expected *InputParameterType !?")
	// 		return
	// 	}
	//
	// 	ipt_nptr := *ipt
	//
	// 	v_str := string(ipt_nptr)
	// 	//if !vok {
	// 	//	err = fmt.Errorf("(HasInputParameterType) expected string !?")
	// 	//	return
	// 	//}
	// 	if v_str == search_type {
	// 		ok = true
	// 		return
	// 	}
	// 	ok = false
	// 	return
	//
	// case *[]InputParameterType:
	// 	array_ipt_ptr, a_ok := array.(*[]InputParameterType)
	// 	if !a_ok {
	// 		err = fmt.Errorf("(HasInputParameterType) expected array A")
	// 		return
	// 	}
	// 	array_ipt := *array_ipt_ptr
	// 	for _, v := range array_ipt {
	// 		v_str := string(v)
	//
	// 		if v_str == search_type {
	// 			ok = true
	// 			return
	// 		}
	// 	}
	//
	// case *[]interface{}:
	// 	array_of_ptr, a_ok := array.(*[]interface{})
	// 	if !a_ok {
	// 		err = fmt.Errorf("(HasInputParameterType) expected array A")
	// 		return
	// 	}
	// 	array_of = *array_of_ptr
	// case []interface{}:
	// 	var a_ok bool
	// 	array_of, a_ok = array.([]interface{})
	// 	if !a_ok {
	// 		err = fmt.Errorf("(HasInputParameterType) expected array A")
	// 		return
	// 	}
	//
	// default:
	// 	err = fmt.Errorf("(HasInputParameterType) expected array B, got %s (search_type: %s)", reflect.TypeOf(array), search_type)
	// 	return
	// }
	//
	// for _, v := range array_of {
	// 	v_str, s_ok := v.(string)
	// 	if !s_ok {
	// 		err = fmt.Errorf("(HasInputParameterType) array element is not string")
	// 		return
	// 	}
	// 	if v_str == search_type {
	// 		ok = true
	// 		return
	// 	}
	// }
	//
	// ok = false
	return
}
