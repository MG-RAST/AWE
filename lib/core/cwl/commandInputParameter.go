package cwl

import (
	"fmt"
	cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
	"reflect"
)

// http://www.commonwl.org/v1.0/CommandLineTool.html#CommandInputParameter

type CommandInputParameter struct {
	Id             string             `yaml:"id,omitempty" bson:"id,omitempty" json:"id,omitempty"`
	SecondaryFiles []string           `yaml:"secondaryFiles,omitempty" bson:"secondaryFiles,omitempty" json:"secondaryFiles,omitempty"` // TODO string | Expression | array<string | Expression>
	Format         []string           `yaml:"format,omitempty" bson:"format,omitempty" json:"format,omitempty"`
	Streamable     bool               `yaml:"streamable,omitempty" bson:"streamable,omitempty" json:"streamable,omitempty"`
	Type           interface{}        `yaml:"type,omitempty" bson:"type,omitempty" json:"type,omitempty"` // []CommandInputParameterType  CWLType | CommandInputRecordSchema | CommandInputEnumSchema | CommandInputArraySchema | string | array<CWLType | CommandInputRecordSchema | CommandInputEnumSchema | CommandInputArraySchema | string>
	Label          string             `yaml:"label,omitempty" bson:"label,omitempty" json:"label,omitempty"`
	Description    string             `yaml:"description,omitempty" bson:"description,omitempty" json:"description,omitempty"`
	InputBinding   CommandLineBinding `yaml:"inputBinding,omitempty" bson:"inputBinding,omitempty" json:"inputBinding,omitempty"`
	Default        *cwl_types.Any     `yaml:"default,omitempty" bson:"default,omitempty" json:"default,omitempty"`
}

func NewCommandInputParameter(v interface{}) (input_parameter *CommandInputParameter, err error) {

	fmt.Println("NewCommandInputParameter:\n")
	spew.Dump(v)

	switch v.(type) {
	case map[interface{}]interface{}:

		v_map, ok := v.(map[interface{}]interface{})
		if !ok {
			err = fmt.Errorf("casting problem (b)")
			return
		}
		v_string_map := make(map[string]interface{})

		for key, value := range v_map {
			key_string := key.(string)
			v_string_map[key_string] = value
		}

		return NewCommandInputParameter(v_string_map)

	case map[string]interface{}:
		v_map, ok := v.(map[string]interface{})

		if !ok {
			err = fmt.Errorf("casting problem (a)")
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

			type_value, err = NewCommandInputParameterType(type_value)
			if err != nil {
				err = fmt.Errorf("(NewCommandInputParameter) NewCommandInputParameter error: %s", err.Error())
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

	case string:
		v_string := v.(string)
		err = fmt.Errorf("(NewCommandInputParameter) got string %s", v_string)
		return
	default:

		err = fmt.Errorf("(NewCommandInputParameter) type %s unknown", reflect.TypeOf(v))
		return
	}

	return
}

// keyname will be converted into 'Id'-field
// array<CommandInputParameter> | map<CommandInputParameter.id, CommandInputParameter.type> | map<CommandInputParameter.id, CommandInputParameter>
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
