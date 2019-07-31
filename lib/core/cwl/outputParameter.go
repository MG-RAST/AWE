package cwl

import (
	"fmt"
	"reflect"

	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
)

// used for ExpressionToolOutputParameter, WorkflowOutputParameter, CommandOutputParameter
type OutputParameter struct {
	Id             string                `yaml:"id,omitempty" bson:"id,omitempty" json:"id,omitempty"`
	Label          string                `yaml:"label,omitempty" bson:"label,omitempty" json:"label,omitempty"`
	SecondaryFiles []Expression          `yaml:"secondaryFiles,omitempty" bson:"secondaryFiles,omitempty" json:"secondaryFiles,omitempty"` // TODO string | Expression | array<string | Expression>
	Format         Expression            `yaml:"format,omitempty" bson:"format,omitempty" json:"format,omitempty"`
	Streamable     bool                  `yaml:"streamable,omitempty" bson:"streamable,omitempty" json:"streamable,omitempty"`
	OutputBinding  *CommandOutputBinding `yaml:"outputBinding,omitempty" bson:"outputBinding,omitempty" json:"outputBinding,omitempty"`
	Type           interface{}           `yaml:"type,omitempty" bson:"type,omitempty" json:"type,omitempty"`
}

// provides Id, Label, SecondaryFiles, Format, Streamable, OutputBinding, Type

// ExpressionToolOutputParameter (context Output)
// CWLType | OutputRecordSchema | OutputEnumSchema | OutputArraySchema | string | array<CWLType | OutputRecordSchema | OutputEnumSchema | OutputArraySchema | string>

// WorkflowOutputParameter (context Output)
// CWLType | OutputRecordSchema | OutputEnumSchema | OutputArraySchema | string | array<CWLType | OutputRecordSchema | OutputEnumSchema | OutputArraySchema | string>

// CommandOutputParameter (context CommandOutput)
// CWLType | stdout | stderr | CommandOutputRecordSchema | CommandOutputEnumSchema | CommandOutputArraySchema | string | array<CWLType | CommandOutputRecordSchema | CommandOutputEnumSchema | CommandOutputArraySchema | string>

// NewOutputParameterFromInterface _
func NewOutputParameterFromInterface(original interface{}, thisID string, schemata []CWLType_Type, context_p string, context *WorkflowContext) (output_parameter *OutputParameter, err error) {
	original, err = MakeStringMap(original, context)
	if err != nil {
		err = fmt.Errorf("(NewOutputParameterFromInterface) MakeStringMap returned: %s", err.Error())
		return
	}

	//fmt.Println("NewOutputParameterFromInterface:")
	//spew.Dump(original)

	switch original.(type) {

	case string:

		// ExpressionTool.Output can be a map of id to type | ExpressionToolOutputParameter
		typeString := original.(string)

		var originalTypeArray []CWLType_Type
		originalTypeArray, err = NewCWLType_TypeFromString(schemata, typeString, "Output")
		if err != nil {
			err = fmt.Errorf("(NewOutputParameterFromInterface) NewCWLType_TypeFromString returned: %s", err.Error())
			return
		}

		output_parameter = &OutputParameter{}
		output_parameter.Id = thisID

		if len(originalTypeArray) == 1 {
			output_parameter.Type = originalTypeArray[0]
		} else {
			output_parameter.Type = originalTypeArray
		}
		//output_parameter.Type = []CWLType_Type{original_type}

	// 	//case int:
	// 	//output_parameter_type, xerr := NewInputParameterTypeArray("int")
	// 	//if xerr != nil {
	// 	//	err = xerr
	// 	//	return
	// 	//}

	// 	//output_parameter.Type = output_parameter_type

	case map[string]interface{}:

		output_parameter = &OutputParameter{}

		originalMap := original.(map[string]interface{})

		//fmt.Println("NewOutputParameterFromInterface as map:")
		//spew.Dump(original_map)

		outputParameterDefault, ok := originalMap["default"]
		if ok {
			originalMap["default"], err = NewCWLType("", "", outputParameterDefault, context)
			if err != nil {
				err = fmt.Errorf("(NewOutputParameterFromInterface) NewCWLType returned: %s", err.Error())
				return
			}
		}

		outputParameterType, ok := originalMap["type"]
		if ok {
			//spew.Dump(outputParameterType)
			//panic("done")
			switch outputParameterType.(type) {
			case []interface{}:
				var outputParameterTypeArray []CWLType_Type
				outputParameterTypeArray, err = NewCWLType_TypeArray(outputParameterType, schemata, context_p, false, context)
				if err != nil {
					err = fmt.Errorf("(NewOutputParameterFromInterface) NewCWLType_TypeArray returned: %s", err.Error())
					return
				}
				if len(outputParameterTypeArray) == 0 {
					err = fmt.Errorf("(NewOutputParameterFromInterface) len(outputParameterTypeArray) == 0")
					return
				}

				if len(outputParameterTypeArray) == 1 {
					originalMap["type"] = outputParameterTypeArray[0]
				} else {
					originalMap["type"] = outputParameterTypeArray
				}
			default:
				var typeArray []CWLType_Type
				typeArray, err = NewCWLType_Type(schemata, outputParameterType, context_p, context)
				if err != nil {
					err = fmt.Errorf("(NewOutputParameterFromInterface) NewCWLType_Type returned: %s", err.Error())
					return
				}
				//typeArray, ok := originalMap["type"].([]CWLType_Type)

				if len(typeArray) == 1 {
					originalMap["type"] = typeArray[0]
				} else {

					originalMap["type"] = typeArray
				}
			}

		}

		outputBinding, hasOutputBinding := originalMap["outputBinding"]
		if hasOutputBinding {
			originalMap["outputBinding"], err = NewCommandOutputBinding(outputBinding, context)
			if err != nil {
				err = fmt.Errorf("(NewOutputParameterFromInterface) NewCommandOutputBinding returns: %s", err.Error())
				return
			}
		}

		err = mapstructure.Decode(original, &output_parameter)
		if err != nil {
			spew.Dump(original)
			err = fmt.Errorf("(NewOutputParameterFromInterface) mapstructure.Decode returned: %s", err.Error())
			return
		}
		if output_parameter.Id == "" {
			output_parameter.Id = thisID

			if output_parameter.Id == "" {
				err = fmt.Errorf("(NewOutputParameterFromInterface) id empty")
				return
			}
		}

	default:
		spew.Dump(original)
		err = fmt.Errorf("(NewOutputParameterFromInterface) cannot parse output type %s", reflect.TypeOf(original))
		return
	}

	//if len(output_parameter.Type) == 0 {
	//	err = fmt.Errorf("(NewOutputParameter) len(output_parameter.Type) == 0")
	//	return
	//}

	return
}

// func NormalizeOutputParameter_deprecated(original_map map[string]interface{}, context *WorkflowContext) (err error) {

// 	outputBinding, ok := original_map["outputBinding"]
// 	if ok {
// 		original_map["outputBinding"], err = NewCommandOutputBinding(outputBinding, context)
// 		if err != nil {
// 			err = fmt.Errorf("(NewCommandOutputParameter) NewCommandOutputBinding returns %s", err.Error())
// 			return
// 		}
// 	}

// 	return
// }

func (op *OutputParameter) IsOptional() (optional bool) {

	switch op.Type.(type) {
	case []interface{}:
		type_array := op.Type.([]interface{})
		for _, my_type := range type_array {
			if my_type == CWLNull {
				optional = true
				return
			}
		}

	}

	optional = false
	return
}
