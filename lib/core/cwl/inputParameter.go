package cwl

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"

	"reflect"
	//"strings"
)

// https://www.commonwl.org/v1.0/Workflow.html#InputParameter

type InputParameter struct {
	CWLObjectImpl  `yaml:",inline" bson:",inline" json:",inline" mapstructure:",squash"` // provides IsCWLObject
	Id             string                                                                `yaml:"id,omitempty" bson:"id,omitempty" json:"id,omitempty" mapstructure:"id,omitempty"`
	Label          string                                                                `yaml:"label,omitempty" bson:"label,omitempty" json:"label,omitempty" mapstructure:"label,omitempty"`
	SecondaryFiles interface{}                                                           `yaml:"secondaryFiles,omitempty" bson:"secondaryFiles,omitempty" json:"secondaryFiles,omitempty" mapstructure:"secondaryFiles,omitempty"` // TODO string | Expression | array<string | Expression>
	Format         []string                                                              `yaml:"format,omitempty" bson:"format,omitempty" json:"format,omitempty" mapstructure:"format,omitempty"`
	Streamable     bool                                                                  `yaml:"streamable,omitempty" bson:"streamable,omitempty" json:"streamable,omitempty" mapstructure:"streamable,omitempty"`
	Doc            string                                                                `yaml:"doc,omitempty" bson:"doc,omitempty" json:"doc,omitempty" mapstructure:"doc,omitempty"`
	InputBinding   *CommandLineBinding                                                   `yaml:"inputBinding,omitempty" bson:"inputBinding,omitempty" json:"inputBinding,omitempty" mapstructure:"inputBinding,omitempty"` //TODO
	Default        CWLType                                                               `yaml:"default,omitempty" bson:"default,omitempty" json:"default,omitempty" mapstructure:"default,omitempty"`
	Type           []CWLType_Type                                                        `yaml:"type,omitempty" bson:"type,omitempty" json:"type,omitempty" mapstructure:"type,omitempty"` // TODO CWLType | InputRecordSchema | InputEnumSchema | InputArraySchema | string | array<CWLType | InputRecordSchema | InputEnumSchema | InputArraySchema | string>
}

func (i InputParameter) GetClass() string { return "InputParameter" }
func (i InputParameter) GetID() string    { return i.Id }
func (i InputParameter) SetID(id string)  { i.Id = id }
func (i InputParameter) IsCWLMinimal()    {}

func NewInputParameter(original interface{}, schemata []CWLType_Type, context *WorkflowContext) (input_parameter *InputParameter, err error) {

	fmt.Println("---- NewInputParameter ----")
	spew.Dump(original)

	original, err = MakeStringMap(original, context)
	if err != nil {
		err = fmt.Errorf("(NewInputParameter) MakeStringMap returned: %s", err.Error())
		return
	}
	//spew.Dump(original)

	input_parameter = &InputParameter{}

	switch original.(type) {
	case string:

		type_string := original.(string)

		var original_type CWLType_Type
		original_type, err = NewCWLType_TypeFromString(schemata, type_string, "Input")
		if err != nil {
			err = fmt.Errorf("(NewInputParameter) NewCWLType_TypeFromString returned: %s", err.Error())
			return
		}

		//input_parameter_type, xerr := NewInputParameterType(type_string_lower)
		//if xerr != nil {
		//	err = xerr
		//	return
		//}

		input_parameter.Type = []CWLType_Type{original_type}

		//case int:
		//input_parameter_type, xerr := NewInputParameterTypeArray("int")
		//if xerr != nil {
		//	err = xerr
		//	return
		//}

		//input_parameter.Type = input_parameter_type

	case map[string]interface{}:

		original_map := original.(map[string]interface{})

		input_parameter_default, ok := original_map["default"]
		if ok {
			original_map["default"], err = NewCWLType("", input_parameter_default, context)
			if err != nil {
				err = fmt.Errorf("(NewInputParameter) NewCWLType returned: %s", err.Error())
				return
			}
		}

		inputParameter_type, ok := original_map["type"]
		if ok {
			var inputParameter_type_array []CWLType_Type
			inputParameter_type_array, err = NewCWLType_TypeArray(inputParameter_type, schemata, "Input", false, context)
			if err != nil {
				err = fmt.Errorf("(NewInputParameter) NewCWLType_TypeArray returns: %s", err.Error())
				return
			}
			if len(inputParameter_type_array) == 0 {
				err = fmt.Errorf("(NewInputParameter) len(inputParameter_type_array) == 0")
				return
			}
			original_map["type"] = inputParameter_type_array
		}

		err = mapstructure.Decode(original, input_parameter)
		if err != nil {
			spew.Dump(original)
			err = fmt.Errorf("(NewInputParameter) mapstructure.Decode returned: %s", err.Error())
			return
		}
	default:
		spew.Dump(original)
		err = fmt.Errorf("(NewInputParameter) cannot parse input type %s", reflect.TypeOf(original))
		return
	}

	if len(input_parameter.Type) == 0 {
		err = fmt.Errorf("(NewInputParameter) len(input_parameter.Type) == 0")
		return
	}

	return
}

// InputParameter
func NewInputParameterArray(original interface{}, schemata []CWLType_Type, context *WorkflowContext) (new_array []InputParameter, err error) {

	original, err = MakeStringMap(original, context)
	if err != nil {
		err = fmt.Errorf("(NewInputParameterArray) MakeStringMap returned: %s", err.Error())
		return
	}

	switch original.(type) {
	case map[string]interface{}:
		original_map := original.(map[string]interface{})
		for id, v := range original_map {
			//fmt.Printf("A")

			input_parameter, xerr := NewInputParameter(v, schemata, context)
			if xerr != nil {
				err = fmt.Errorf("(NewInputParameterArray) A NewInputParameter returned: %s", xerr.Error())
				return
			}

			input_parameter.Id = id

			if input_parameter.Id == "" {
				err = fmt.Errorf("(NewInputParameterArray) ID is missing")
				return
			}

			//fmt.Printf("C")
			new_array = append(new_array, *input_parameter)
			//fmt.Printf("D")

		}
	case []interface{}:
		original_array := original.([]interface{})
		for _, v := range original_array {
			//fmt.Printf("A")

			input_parameter, xerr := NewInputParameter(v, schemata, context)
			if xerr != nil {
				err = fmt.Errorf("(NewInputParameterArray) B NewInputParameter returned: %s", xerr.Error())
				return
			}

			if input_parameter.Id == "" {
				err = fmt.Errorf("(NewInputParameterArray) ID is missing")
				return
			}

			//fmt.Printf("C")
			new_array = append(new_array, *input_parameter)
			//fmt.Printf("D")

		}
	default:
		spew.Dump(original)
		err = fmt.Errorf("(NewInputParameterArray) type %s unknown", reflect.TypeOf(original))
		return
	}

	//spew.Dump(new_array)
	//os.Exit(0)
	return
}
