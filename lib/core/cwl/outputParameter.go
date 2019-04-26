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

		var originalType CWLType_Type
		originalType, err = NewCWLType_TypeFromString(schemata, typeString, "Output")
		if err != nil {
			err = fmt.Errorf("(NewInputParameter) NewCWLType_TypeFromString returned: %s", err.Error())
			return
		}

		output_parameter = &OutputParameter{}
		output_parameter.Id = thisID
		output_parameter.Type = originalType

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

		original_map := original.(map[string]interface{})

		//fmt.Println("NewOutputParameterFromInterface as map:")
		//spew.Dump(original_map)

		output_parameter_default, ok := original_map["default"]
		if ok {
			original_map["default"], err = NewCWLType("", output_parameter_default, context)
			if err != nil {
				err = fmt.Errorf("(NewOutputParameterFromInterface) NewCWLType returned: %s", err.Error())
				return
			}
		}

		outputParameter_type, ok := original_map["type"]
		if ok {

			switch outputParameter_type.(type) {
			case []interface{}:
				var outputParameter_type_array []CWLType_Type
				outputParameter_type_array, err = NewCWLType_TypeArray(outputParameter_type, schemata, context_p, false, context)
				if err != nil {
					err = fmt.Errorf("(NewOutputParameterFromInterface) NewCWLType_TypeArray returned: %s", err.Error())
					return
				}
				if len(outputParameter_type_array) == 0 {
					err = fmt.Errorf("(NewOutputParameterFromInterface) len(outputParameter_type_array) == 0")
					return
				}
				original_map["type"] = outputParameter_type_array
			default:
				original_map["type"], err = NewCWLType_Type(schemata, outputParameter_type, context_p, context)
				if err != nil {
					err = fmt.Errorf("(NewOutputParameterFromInterface) NewCWLType_Type returned: %s", err.Error())
					return
				}
			}

		}

		outputBinding, has_outputBinding := original_map["outputBinding"]
		if has_outputBinding {
			original_map["outputBinding"], err = NewCommandOutputBinding(outputBinding, context)
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

func NormalizeOutputParameter_deprecated(original_map map[string]interface{}, context *WorkflowContext) (err error) {

	outputBinding, ok := original_map["outputBinding"]
	if ok {
		original_map["outputBinding"], err = NewCommandOutputBinding(outputBinding, context)
		if err != nil {
			err = fmt.Errorf("(NewCommandOutputParameter) NewCommandOutputBinding returns %s", err.Error())
			return
		}
	}

	return
}

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
