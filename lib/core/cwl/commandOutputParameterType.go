package cwl

import (
	"fmt"
	"reflect"

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
	if err != nil {

		fmt.Println("NewCommandOutputParameterType:")
		spew.Dump(original)

		err = fmt.Errorf("(NewCommandOutputParameterType) NewCWLType_Type (context CommandOutput) returned: %s", err.Error())
		return
	}

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

		var copt interface{}
		copt, err = NewCommandOutputParameterType(original, schemata)
		if err != nil {
			err = fmt.Errorf("(NewCommandOutputParameterTypeArray) A) NewCommandOutputParameterType returned: %s", err.Error())
			return
		}
		result_array = append(result_array, copt)
		return

	case []interface{}:
		logger.Debug(3, "[found array]")

		original_array := original.([]interface{})

		for _, element := range original_array {

			//spew.Dump(original)
			var copt interface{}
			copt, err = NewCommandOutputParameterType(element, schemata)
			if err != nil {

				fmt.Println("NewCommandOutputParameterTypeArray:")
				spew.Dump(original_array)

				err = fmt.Errorf("(NewCommandOutputParameterTypeArray) B) NewCommandOutputParameterType returned: %s", err.Error())
				return
			}
			result_array = append(result_array, copt)
		}

		return

	case string:

		var copt interface{}
		copt, err = NewCommandOutputParameterType(original, schemata)
		if err != nil {
			err = fmt.Errorf("(NewCommandOutputParameterTypeArray) C) NewCommandOutputParameterType returned: %s", err.Error())
			return
		}
		result_array = append(result_array, copt)

		return
	default:
		fmt.Printf("unknown type:\n")
		spew.Dump(original)
		err = fmt.Errorf("(NewCommandOutputParameterTypeArray) type not supported (%s)", reflect.TypeOf(original))
	}
	return

}
