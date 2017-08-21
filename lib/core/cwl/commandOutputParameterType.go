package cwl

import (
	"fmt"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
	//"github.com/mitchellh/mapstructure"
	//"strings"
	cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	"reflect"
)

//type of http://www.commonwl.org/v1.0/CommandLineTool.html#CommandOutputParameter

// CWLType | stdout | stderr | CommandOutputRecordSchema | CommandOutputEnumSchema | CommandOutputArraySchema | string |
// array<CWLType | CommandOutputRecordSchema | CommandOutputEnumSchema | CommandOutputArraySchema | string>

type CommandOutputParameterType interface {
	Is_CommandOutputParameterType()
}

type CommandOutputParameterTypeImpl struct {
	Type string
}

func (c *CommandOutputParameterTypeImpl) Is_CommandOutputParameterType() {}

func NewCommandOutputParameterType(original interface{}) (copt_ptr *CommandOutputParameterType, err error) {

	// Try CWL_Type
	var copt CommandOutputParameterTypeImpl

	switch original.(type) {

	case string:
		original_str := original.(string)

		//original_str_lower := strings.ToLower(original_str)

		_, is_valid := cwl_types.Valid_cwltypes[original_str]

		if !is_valid {
			err = fmt.Errorf("(NewCommandOutputParameterType) type %s is unknown", original_str)
			return
		}

		copt.Type = original_str
		copt_ptr = &copt
		return

	case map[interface{}]interface{}:
		original_map, ok := original.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("type error")
			return
		}
		return NewCommandOutputParameterType(original_map)
	case map[string]interface{}:
		// CommandOutputArraySchema www.commonwl.org/v1.0/CommandLineTool.html#CommandOutputArraySchema
		original_map, ok := original.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("type error")
			return
		}
		output_type, ok := original_map["type"]

		if !ok {
			fmt.Printf("unknown type")
			spew.Dump(original)
			err = fmt.Errorf("(NewCommandOutputParameterType) map[string]interface{} has no type field")
			return
		}

		switch output_type {
		case "array":
			copt.CommandOutputArraySchema, err = NewCommandOutputArraySchema(original_map)
			copt_ptr = &copt
			return
		default:
			fmt.Printf("unknown type %s:", output_type)
			spew.Dump(original)
			err = fmt.Errorf("(NewCommandOutputParameterType) Map-Type %s unknown", reflect.TypeOf(output_type))
			return
		}

	}

	spew.Dump(original)
	err = fmt.Errorf("(NewCommandOutputParameterType) Type %s unknown", reflect.TypeOf(original))

	return

}

func NewCommandOutputParameterTypeArray(original interface{}) (copta *[]CommandOutputParameterType, err error) {

	switch original.(type) {
	case map[interface{}]interface{}:
		logger.Debug(3, "[found map]")

		copt, xerr := NewCommandOutputParameterType(original)
		if xerr != nil {
			err = xerr
			return
		}
		copta = &[]CommandOutputParameterType{*copt}
	case []interface{}:
		logger.Debug(3, "[found array]")

		copta_nptr := []CommandOutputParameterType{}

		original_array := original.([]interface{})

		for _, element := range original_array {

			spew.Dump(original)
			copt, xerr := NewCommandOutputParameterType(element)
			if xerr != nil {
				err = xerr
				return
			}
			copta_nptr = append(copta_nptr, *copt)
		}

		copta = &copta_nptr
	case string:
		copta_nptr := []CommandOutputParameterType{}

		copt, xerr := NewCommandOutputParameterType(original)
		if xerr != nil {
			err = xerr
			return
		}
		copta_nptr = append(copta_nptr, *copt)

		copta = &copta_nptr
	default:
		fmt.Printf("unknown type")
		spew.Dump(original)
		err = fmt.Errorf("(NewCommandOutputParameterTypeArray) unknown type")
	}
	return

}
