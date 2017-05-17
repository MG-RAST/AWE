package cwl

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
	"strings"
)

type InputParameter struct {
	Id             string                `yaml:"id"`
	Label          string                `yaml:"label"`
	SecondaryFiles []string              `yaml:"secondaryFiles"` // TODO string | Expression | array<string | Expression>
	Format         string                `yaml:"format"`
	Streamable     bool                  `yaml:"streamable"`
	Doc            string                `yaml:"doc"`
	InputBinding   CommandLineBinding    `yaml:"inputBinding"` //TODO
	Default        Any                   `yaml:"default"`
	Type           *[]InputParameterType `yaml:"type"` // TODO CWLType | InputRecordSchema | InputEnumSchema | InputArraySchema | string | array<CWLType | InputRecordSchema | InputEnumSchema | InputArraySchema | string>
}

func (i InputParameter) GetClass() string { return "InputParameter" }
func (i InputParameter) GetId() string    { return i.Id }
func (i InputParameter) SetId(id string)  { i.Id = id }
func (i InputParameter) is_CWL_minimal()  {}

func NewInputParameter(original interface{}) (input_parameter *InputParameter, err error) {

	input_parameter = &InputParameter{}

	switch original.(type) {
	case string:

		type_string := original.(string)
		type_string_lower := strings.ToLower(type_string)

		switch type_string_lower {
		case "string":
		case "int":
		case "file":
		default:
			err = fmt.Errorf("unknown type: \"%s\"", type_string)
			return
		}

		input_parameter_type, xerr := NewInputParameterTypeArray(type_string_lower)
		if xerr != nil {
			err = xerr
			return
		}

		input_parameter.Type = input_parameter_type

		//case int:
		//input_parameter_type, xerr := NewInputParameterTypeArray("int")
		//if xerr != nil {
		//	err = xerr
		//	return
		//}

		//input_parameter.Type = input_parameter_type
	case map[interface{}]interface{}:

		original_map := original.(map[interface{}]interface{})

		input_parameter_default, ok := original_map["default"]
		if ok {
			original_map["default"], err = NewAny(input_parameter_default)
			if err != nil {
				return
			}
		}

		inputParameter_type, ok := original_map["type"]
		if ok {
			original_map["type"], err = NewInputParameterTypeArray(inputParameter_type)
			if err != nil {
				return
			}
		}

		err = mapstructure.Decode(original, input_parameter)
		if err != nil {
			err = fmt.Errorf("(NewInputParameter) %s", err.Error())
			return
		}
	default:
		err = fmt.Errorf("(NewInputParameter) cannot parse input")
		return
	}

	return
}

// InputParameter
func NewInputParameterArray(original interface{}) (err error, new_array []InputParameter) {

	switch original.(type) {
	case map[interface{}]interface{}:
		original_map := original.(map[interface{}]interface{})
		for k, v := range original_map {
			//fmt.Printf("A")

			id, ok := k.(string)
			if !ok {
				err = fmt.Errorf("Cannot parse id of input")
				return
			}

			input_parameter, xerr := NewInputParameter(v)
			if xerr != nil {
				err = xerr
				return
			}

			input_parameter.Id = id

			if input_parameter.Id == "" {
				err = fmt.Errorf("ID is missing")
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

			input_parameter, xerr := NewInputParameter(v)
			if xerr != nil {
				err = xerr
				return
			}

			if input_parameter.Id == "" {
				err = fmt.Errorf("ID is missing")
				return
			}

			//fmt.Printf("C")
			new_array = append(new_array, *input_parameter)
			//fmt.Printf("D")

		}
	default:
		spew.Dump(original)
		err = fmt.Errorf("(NewInputParameterArray) type unknown")
		return
	}

	//spew.Dump(new_array)
	//os.Exit(0)
	return
}
