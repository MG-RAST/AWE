package cwl

import (
	"fmt"
	cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	"github.com/davecgh/go-spew/spew"
	"strings"
	//"github.com/mitchellh/mapstructure"
	"reflect"
)

type CommandInputParameterType struct {
	Type string
}

// CWLType | CommandInputRecordSchema | CommandInputEnumSchema | CommandInputArraySchema | string | array<CWLType | CommandInputRecordSchema | CommandInputEnumSchema | CommandInputArraySchema | string>

func NewCommandInputParameterType(original interface{}) (result interface{}, err error) {

	// Try CWL_Type
	//var cipt CommandInputParameterType

	switch original.(type) {

	case []interface{}:

	case map[string]interface{}:

		original_map, ok := original.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewCommandInputParameterType) type error")
			return
		}

		type_str, ok := original_map["Type"]
		if !ok {
			type_str, ok = original_map["type"]
		}

		if !ok {
			err = fmt.Errorf("(NewCommandInputParameterType) type error, field type not found")
			return
		}

		switch type_str {
		case "array":
			schema, xerr := NewCommandOutputArraySchema(original_map)
			if xerr != nil {
				err = xerr
				return
			}
			result = schema
			return
		case "enum":

			schema, xerr := NewCommandOutputEnumSchema(original_map)
			if xerr != nil {
				err = xerr
				return
			}
			result = schema
			return

		case "record":
			schema, xerr := NewCommandOutputRecordSchema(original_map)
			if xerr != nil {
				err = xerr
				return
			}
			result = schema
			return

		}
		err = fmt.Errorf("(NewCommandInputParameterType) type %s unknown", type_str)
		return

	case string:
		original_str := original.(string)

		switch strings.ToLower(original_str) {

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
			err = fmt.Errorf("(NewCommandInputParameterType) type %s is unknown", original_str)
			return
		}
		result = original_str
		return

	}

	fmt.Printf("unknown type")
	spew.Dump(original)
	err = fmt.Errorf("(NewCommandInputParameterType) Type %s unknown", reflect.TypeOf(original))

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
