package cwl

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
)

type CommandInputParameter struct {
	Id             string                      `yaml:"id"`
	SecondaryFiles []string                    `yaml:"secondaryFiles"` // TODO string | Expression | array<string | Expression>
	Format         string                      `yaml:"format"`
	Streamable     bool                        `yaml:"streamable"`
	Type           []CommandInputParameterType `yaml:"type"` // TODO CWLType | CommandInputRecordSchema | CommandInputEnumSchema | CommandInputArraySchema | string | array<CWLType | CommandInputRecordSchema | CommandInputEnumSchema | CommandInputArraySchema | string>
	Label          string                      `yaml:"label"`
	Description    string                      `yaml:"description"`
	InputBinding   CommandLineBinding          `yaml:"inputBinding"`
	Default        Any                         `yaml:"default"`
}

type CommandInputParameterType interface {
	CWL_minimal_interface
	is_CommandInputParameterType()
}

type CommandInputParameterType_Impl struct{}

func (c *CommandInputParameterType_Impl) is_CommandInputParameterType() {}

func NewCommandInputParameterType(unparsed interface{}) (cipt_ptr *CommandInputParameterType, err error) {

	// Try CWL_Type
	var cipt CommandInputParameterType

	cwl_type, err := NewCWLType(unparsed) // returns *CWLType_Impl

	if err == nil {

		//cwl_type_x := *cwl_type

		cipt = cwl_type.(CommandInputParameterType)
		cipt_ptr = &cipt
		return
	}

	err = fmt.Errorf("(NewCommandInputParameterType) Type unknown")

	return

}

func NewCommandInputParameter(v interface{}) (input_parameter *CommandInputParameter, err error) {

	switch v.(type) {
	case map[interface{}]interface{}:
		v_map := v.(map[interface{}]interface{})

		default_value, ok := v_map["default"]
		if ok {
			v_map["default"], err = NewAny(default_value) // TODO return Int or similar
			if err != nil {
				return
			}
		}

		type_value, ok := v_map["type"]
		if ok {
			v_map["type"], err = CreateCommandInputParameterTypeArray(type_value)
			if err != nil {
				return
			}
		}

		input_parameter = &CommandInputParameter{}
		err = mapstructure.Decode(v, input_parameter)

	default:
		err = fmt.Errorf("(NewCommandInputParameter) type unknown")
		return
	}

	return
}

func CreateCommandInputParameterTypeArray(v interface{}) (cipt_array_ptr *[]CommandInputParameterType, err error) {

	cipt_array := []CommandInputParameterType{}

	array, ok := v.([]interface{})

	if ok {
		//handle array case
		for _, v := range array {

			cipt, xerr := NewCommandInputParameterType(v)
			if xerr != nil {
				err = xerr
				return
			}

			cipt_array = append(cipt_array, *cipt)
		}
		cipt_array_ptr = &cipt_array
		return
	}

	// handle non-arrary case

	cipt, err := NewCommandInputParameterType(v)
	if err != nil {

		return
	}

	cipt_array = append(cipt_array, *cipt)
	cipt_array_ptr = &cipt_array

	return
}

// keyname will be converted into 'Id'-field
func CreateCommandInputArray(original interface{}) (err error, new_array []*CommandInputParameter) {

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
