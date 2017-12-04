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

func NewCommandOutputParameterType(original interface{}) (result interface{}, err error) {

	original, err = MakeStringMap(original)
	if err != nil {
		return
	}

	result, err = NewCWLType_Type(original, "CommandOutput")

	// switch original.(type) {
	//
	// 	case string:
	// 		original_str := original.(string)
	//
	// 		//original_str_lower := strings.ToLower(original_str)
	//
	// 		the_type, is_valid := IsValidType(original_str)
	//
	// 		if !is_valid {
	// 			err = fmt.Errorf("(NewCommandOutputParameterType) type %s is unknown", original_str)
	// 			return
	// 		}
	//
	// 		result = the_type
	// 		return
	//
	// 	case map[string]interface{}:
	// 		// CommandOutputArraySchema www.commonwl.org/v1.0/CommandLineTool.html#CommandOutputArraySchema
	//
	//
	// 		NewCWLType_Type(original)
	//
	// 		original_map, ok := original.(map[string]interface{})
	// 		if !ok {
	// 			err = fmt.Errorf("type error")
	// 			return
	// 		}
	// 		output_type, ok := original_map["type"]
	//
	// 		if !ok {
	// 			fmt.Printf("unknown type")
	// 			spew.Dump(original)
	// 			err = fmt.Errorf("(NewCommandOutputParameterType) map[string]interface{} has no type field")
	// 			return
	// 		}
	//
	// 		switch output_type {
	// 		case "array":
	// 			copt.CommandOutputArraySchema, err = NewCommandOutputArraySchema(original_map)
	// 			copt_ptr = &copt
	// 			return
	// 		default:
	// 			fmt.Printf("unknown type %s:", output_type)
	// 			spew.Dump(original)
	// 			err = fmt.Errorf("(NewCommandOutputParameterType) Map-Type %s unknown", reflect.TypeOf(output_type))
	// 			return
	// 		}
	//
	// 	}
	//
	// 	spew.Dump(original)
	// 	err = fmt.Errorf("(NewCommandOutputParameterType) Type %s unknown", reflect.TypeOf(original))

	return

}

func NewCommandOutputParameterTypeArray(original interface{}) (result_array []interface{}, err error) {

	original, err = MakeStringMap(original)
	if err != nil {
		return
	}

	result_array = []interface{}{}

	switch original.(type) {
	case map[string]interface{}:
		logger.Debug(3, "[found map]")

		copt, xerr := NewCommandOutputParameterType(original)
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
			copt, xerr := NewCommandOutputParameterType(element)
			if xerr != nil {
				err = xerr
				return
			}
			result_array = append(result_array, copt)
		}

		return

	case string:

		copt, xerr := NewCommandOutputParameterType(original)
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
