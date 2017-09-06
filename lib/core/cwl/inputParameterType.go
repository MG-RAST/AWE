package cwl

import (
	"fmt"
	cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	"github.com/davecgh/go-spew/spew"
	"reflect"
	"strings"
)

//type InputParameterType struct {
//	Type string
//}

type InputParameterType string

func NewInputParameterType(original interface{}) (ipt_ptr *InputParameterType, err error) {
	fmt.Println("---- NewInputParameterType ----")

	spew.Dump(original)
	var ipt InputParameterType

	switch original.(type) {

	case string:
		original_str := original.(string)

		original_str_lower := strings.ToLower(original_str)
		switch original_str_lower {

		case cwl_types.CWL_null:
		case cwl_types.CWL_boolean:
		case cwl_types.CWL_int:
		case cwl_types.CWL_long:
		case cwl_types.CWL_float:
		case cwl_types.CWL_double:
		case cwl_types.CWL_string:
		case strings.ToLower(cwl_types.CWL_File):
		case strings.ToLower(cwl_types.CWL_Directory):
		default:
			err = fmt.Errorf("(NewInputParameterType) type %s is unknown", original_str_lower)
			return
		}

		ipt = InputParameterType(original_str)

		ipt_ptr = &ipt
		return
	default:
		err = fmt.Errorf("(NewInputParameterType) type is not string: %s", reflect.TypeOf(original))
		return
	}

	return
}

func NewInputParameterTypeArray(original interface{}) (array_ptr *[]InputParameterType, err error) {
	fmt.Println("---- NewInputParameterTypeArray ----")
	spew.Dump(original)
	array := []InputParameterType{}

	switch original.(type) {
	case []interface{}:
		original_array := original.([]interface{})
		for _, v := range original_array {
			ipt, xerr := NewInputParameterType(v)
			if xerr != nil {
				err = xerr
				return
			}
			array = append(array, *ipt)
		}
		array_ptr = &array
		return
	case string:
		ipt, xerr := NewInputParameterType(original)
		if xerr != nil {
			err = xerr
			return
		}
		array = append(array, *ipt)
		array_ptr = &array
		return
	default:

		err = fmt.Errorf("(NewInputParameterTypeArray) type unknown")
		return

	}
}

func HasInputParameterType(array interface{}, search_type string) (ok bool, err error) {

	var array_of []interface{}

	switch array.(type) {
	case *[]InputParameterType:
		array_ipt_ptr, a_ok := array.(*[]InputParameterType)
		if !a_ok {
			err = fmt.Errorf("(HasInputParameterType) expected array A")
			return
		}
		array_ipt := *array_ipt_ptr
		for _, v := range array_ipt {
			v_str := string(v)

			if v_str == search_type {
				ok = true
				return
			}
		}

	case *[]interface{}:
		array_of_ptr, a_ok := array.(*[]interface{})
		if !a_ok {
			err = fmt.Errorf("(HasInputParameterType) expected array A")
			return
		}
		array_of = *array_of_ptr
	case []interface{}:
		var a_ok bool
		array_of, a_ok = array.([]interface{})
		if !a_ok {
			err = fmt.Errorf("(HasInputParameterType) expected array A")
			return
		}

	default:
		err = fmt.Errorf("(HasInputParameterType) expected array B, got %s", reflect.TypeOf(array))
		return
	}

	for _, v := range array_of {
		v_str, s_ok := v.(string)
		if !s_ok {
			err = fmt.Errorf("(HasInputParameterType) array element is not string")
			return
		}
		if v_str == search_type {
			ok = true
			return
		}
	}

	ok = false
	return
}
