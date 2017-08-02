package cwl

import (
	"fmt"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
	//"strings"
	cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
)

//type of http://www.commonwl.org/v1.0/CommandLineTool.html#CommandOutputParameter

type CommandOutputParameterType struct {
	Type                      string
	CommandOutputArraySchema  *CommandOutputArraySchema
	CommandOutputRecordSchema *CommandOutputRecordSchema
}

type CommandOutputRecordSchema struct {
	Type   string                     `yaml:"type" bson:"type" json:"type"` // Must be record
	Fields []CommandOutputRecordField `yaml:"fields" bson:"fields" json:"fields"`
	Label  string                     `yaml:"label" bson:"label" json:"label"`
}

type CommandOutputRecordField struct{}

//http://www.commonwl.org/v1.0/CommandLineTool.html#CommandOutputEnumSchema
type CommandOutputEnumSchema struct {
	Symbols       []string              `yaml:"symbols" bson:"symbols" json:"symbols"`
	Type          string                `yaml:"type" bson:"type" json:"type"` // must be enum
	Label         string                `yaml:"label" bson:"label" json:"label"`
	OutputBinding *CommandOutputBinding `yaml:"outputbinding" bson:"outputbinding" json:"outputbinding"`
}

type CommandOutputArraySchema struct {
	Items         []string              `yaml:"items" bson:"items" json:"items"`
	Type          string                `yaml:"type" bson:"type" json:"type"` // must be array
	Label         string                `yaml:"label" bson:"label" json:"label"`
	OutputBinding *CommandOutputBinding `yaml:"outputBinding" bson:"outputBinding" json:"outputBinding"`
}

func NewCommandOutputArraySchema(original map[interface{}]interface{}) (coas *CommandOutputArraySchema, err error) {
	coas = &CommandOutputArraySchema{}

	items, ok := original["items"]
	if ok {
		items_string, ok := items.(string)
		if ok {
			original["items"] = []string{items_string}
		}
	}

	err = mapstructure.Decode(original, coas)
	if err != nil {
		err = fmt.Errorf("(NewCommandOutputArraySchema) %s", err.Error())
		return
	}
	return
}

func NewCommandOutputParameterType(original interface{}) (copt_ptr *CommandOutputParameterType, err error) {

	// Try CWL_Type
	var copt CommandOutputParameterType

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
		// CommandOutputArraySchema www.commonwl.org/v1.0/CommandLineTool.html#CommandOutputArraySchema
		original_map := original.(map[interface{}]interface{})

		output_type, ok := original_map["type"]

		if !ok {
			fmt.Printf("unknown type")
			spew.Dump(original)
			err = fmt.Errorf("(NewCommandOutputParameterType) Map-Type unknown")
		}

		switch output_type {
		case "array":
			copt.CommandOutputArraySchema, err = NewCommandOutputArraySchema(original_map)
			copt_ptr = &copt
			return
		default:
			fmt.Printf("unknown type %s:", output_type)
			spew.Dump(original)
			err = fmt.Errorf("(NewCommandOutputParameterType) Map-Type %s unknown", output_type)
			return
		}

	}

	fmt.Printf("unknown type")
	spew.Dump(original)
	err = fmt.Errorf("(NewCommandOutputParameterType) Type unknown")

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
