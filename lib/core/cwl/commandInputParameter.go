package cwl

import (
	"fmt"
	cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
)

type CommandInputParameter struct {
	Id             string                      `yaml:"id"`
	SecondaryFiles []string                    `yaml:"secondaryFiles"` // TODO string | Expression | array<string | Expression>
	Format         []string                    `yaml:"format"`
	Streamable     bool                        `yaml:"streamable"`
	Type           []CommandInputParameterType `yaml:"type"` // TODO CWLType | CommandInputRecordSchema | CommandInputEnumSchema | CommandInputArraySchema | string | array<CWLType | CommandInputRecordSchema | CommandInputEnumSchema | CommandInputArraySchema | string>
	Label          string                      `yaml:"label"`
	Description    string                      `yaml:"description"`
	InputBinding   CommandLineBinding          `yaml:"inputBinding"`
	Default        *cwl_types.Any              `yaml:"default"`
}

func NewCommandInputParameter(v interface{}) (input_parameter *CommandInputParameter, err error) {

	fmt.Println("NewCommandInputParameter:\n")
	spew.Dump(v)

	switch v.(type) {
	case map[interface{}]interface{}:
		v_map, ok := v.(map[interface{}]interface{})

		if !ok {
			err = fmt.Errorf("casting problem")
			return
		}

		default_value, ok := v_map["default"]
		if ok {
			fmt.Println("FOUND default key")
			default_input, xerr := cwl_types.NewAny(default_value) // TODO return Int or similar
			if xerr != nil {
				err = xerr
				return
			}
			spew.Dump(default_input)
			v_map["default"] = default_input
		} else {
			fmt.Println("FOUND NOT default key")
		}

		type_value, ok := v_map["type"]
		if ok {
			type_value, err = CreateCommandInputParameterTypeArray(type_value)
			if err != nil {
				err = fmt.Errorf("(NewCommandInputParameter) CreateCommandInputParameterTypeArray error: %s", err.Error())
				return
			}
			v_map["type"] = type_value
		}

		format_value, ok := v_map["format_value"]
		if ok {
			format_str, is_string := format_value.(string)
			if is_string {
				v_map["format_value"] = []string{format_str}
			}

		}

		input_parameter = &CommandInputParameter{}
		err = mapstructure.Decode(v, input_parameter)
		if err != nil {
			err = fmt.Errorf("(NewCommandInputParameter) decode error: %s", err.Error())
			return
		}

	default:

		err = fmt.Errorf("(NewCommandInputParameter) type unknown")
		return
	}

	return
}

// keyname will be converted into 'Id'-field
func CreateCommandInputArray(original interface{}) (err error, new_array []*CommandInputParameter) {

	fmt.Println("CreateCommandInputArray:\n")
	spew.Dump(original)

	switch original.(type) {
	case map[interface{}]interface{}:
		for k, v := range original.(map[interface{}]interface{}) {

			//var input_parameter CommandInputParameter
			//mapstructure.Decode(v, &input_parameter)
			input_parameter, xerr := NewCommandInputParameter(v)
			if xerr != nil {
				err = fmt.Errorf("(CreateCommandInputArray) map[interface{}]interface{} NewCommandInputParameter returned: %s", xerr)
				return
			}

			input_parameter.Id = k.(string)

			//fmt.Printf("C")
			new_array = append(new_array, input_parameter)
			//fmt.Printf("D")

		}
	case []interface{}:
		for _, v := range original.([]interface{}) {

			input_parameter, xerr := NewCommandInputParameter(v)
			if xerr != nil {
				err = fmt.Errorf("(CreateCommandInputArray) []interface{} NewCommandInputParameter returned: %s", xerr)
				return
			}
			//var input_parameter CommandInputParameter
			//mapstructure.Decode(v, &input_parameter)

			//empty, err := NewEmpty(v)
			//if err != nil {
			//	return
			//}

			if input_parameter.Id == "" {
				err = fmt.Errorf("(CreateCommandInputArray) no id found")
				return
			}

			//fmt.Printf("C")
			new_array = append(new_array, input_parameter)
			//fmt.Printf("D")

		}
	default:
		err = fmt.Errorf("(CreateCommandInputArray) type unknown")
	}
	//spew.Dump(new_array)
	return
}
