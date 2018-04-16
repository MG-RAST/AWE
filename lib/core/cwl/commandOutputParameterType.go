package cwl

import (
	"fmt"

	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
	//"github.com/mitchellh/mapstructure"
	//"strings"
	//"reflect"
)

//type of http://www.commonwl.org/v1.0/CommandLineTool.html#CommandOutputParameter

// CWLType | stdout | stderr | CommandOutputRecordSchema | CommandOutputEnumSchema | CommandOutputArraySchema | string |
// array<CWLType | CommandOutputRecordSchema | CommandOutputEnumSchema | CommandOutputArraySchema | string>

func NewCommandOutputParameterType(original interface{}, schemata []CWLType_Type) (result interface{}, err error) {

	original, err = MakeStringMap(original)
	if err != nil {
		return
	}

	result, err = NewCWLType_Type(schemata, original, "CommandOutput")

	return

}

func NewCommandOutputParameterTypeArray(original interface{}, schemata []CWLType_Type) (result_array []interface{}, err error) {

	original, err = MakeStringMap(original)
	if err != nil {
		return
	}

	result_array = []interface{}{}

	switch original.(type) {
	case map[string]interface{}:
		logger.Debug(3, "[found map]")

		copt, xerr := NewCommandOutputParameterType(original, schemata)
		if xerr != nil {
			err = xerr
			return
		}
		result_array = append(result_array, copt)
		return

	case []interface{}:
		logger.Debug(3, "[found array]")

		original_array := original.([]interface{})

		for _, element := range original_array {

			//spew.Dump(original)
			copt, xerr := NewCommandOutputParameterType(element, schemata)
			if xerr != nil {
				err = xerr
				return
			}
			result_array = append(result_array, copt)
		}

		return

	case string:

		copt, xerr := NewCommandOutputParameterType(original, schemata)
		if xerr != nil {
			err = xerr
			return
		}
		result_array = append(result_array, copt)

		return
	default:
		fmt.Printf("unknown type")
		spew.Dump(original)
		err = fmt.Errorf("(NewCommandOutputParameterTypeArray) unknown type")
	}
	return

}
