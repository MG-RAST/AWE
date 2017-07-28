package cwl

import (
	"fmt"
	cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	"strings"
)

type InputParameterType struct {
	Type string
}

func NewInputParameterType(original interface{}) (ipt_ptr *InputParameterType, err error) {

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

		ipt.Type = original_str
		ipt_ptr = &ipt
		return
	default:
	}

	return
}

func NewInputParameterTypeArray(original interface{}) (array_ptr *[]InputParameterType, err error) {

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

func HasInputParameterType(array *[]InputParameterType, search_type string) (ok bool) {
	for _, v := range *array {
		if v.Type == search_type {
			return true
		}
	}
	return false
}
