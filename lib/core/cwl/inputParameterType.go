package cwl

import (
	"fmt"
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

		case "null":
		case "boolean":
		case CWL_int:
		case "long":
		case "float":
		case "double":
		case "string":
		case "file":
		case "directory":
		default:
			err = fmt.Errorf("type %s is unknown", original_str_lower)
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
