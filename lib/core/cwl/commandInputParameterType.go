package cwl

import (
	"fmt"
	cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	"github.com/davecgh/go-spew/spew"
	//"strings"
	//"github.com/mitchellh/mapstructure"
)

type CommandInputParameterType struct {
	Type string
}

func NewCommandInputParameterType(original interface{}) (cipt_ptr *CommandInputParameterType, err error) {

	// Try CWL_Type
	var cipt CommandInputParameterType

	switch original.(type) {

	case string:
		original_str := original.(string)

		switch original_str {

		case cwl_types.CWL_null:
		case cwl_types.CWL_boolean:
		case cwl_types.CWL_int:
		case cwl_types.CWL_long:
		case cwl_types.CWL_float:
		case cwl_types.CWL_double:
		case cwl_types.CWL_string:
		case cwl_types.CWL_File:
		case cwl_types.CWL_Directory:
		default:
			err = fmt.Errorf("(NewCommandInputParameterType) type %s is unknown", original_str)
			return
		}

		cipt.Type = original_str
		cipt_ptr = &cipt
		return

	}

	fmt.Printf("unknown type")
	spew.Dump(original)
	err = fmt.Errorf("(NewCommandInputParameterType) Type unknown")

	return

}

func CreateCommandInputParameterTypeArray(v interface{}) (cipt_array_ptr *[]CommandInputParameterType, err error) {

	cipt_array := []CommandInputParameterType{}

	array, ok := v.([]interface{})

	if ok {
		//handle array case
		for _, v := range array {

			cipt, xerr := NewCommandInputParameterType(v)
			if xerr != nil {
				err = xerr
				return
			}

			cipt_array = append(cipt_array, *cipt)
		}
		cipt_array_ptr = &cipt_array
		return
	}

	// handle non-arrary case

	cipt, err := NewCommandInputParameterType(v)
	if err != nil {
		err = fmt.Errorf("(CreateCommandInputParameterTypeArray) NewCommandInputParameterType returns %s", err.Error())
		return
	}

	cipt_array = append(cipt_array, *cipt)
	cipt_array_ptr = &cipt_array

	return
}

func HasCommandInputParameterType(array *[]CommandInputParameterType, search_type string) (ok bool) {
	for _, v := range *array {
		if v.Type == search_type {
			return true
		}
	}
	return false
}
