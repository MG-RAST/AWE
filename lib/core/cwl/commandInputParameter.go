package cwl

import (
	"fmt"
	"reflect"

	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
)

// CommandInputParameter http://www.commonwl.org/v1.0/CommandLineTool.html#CommandInputParameter
type CommandInputParameter struct {
	CWLObjectImpl  `yaml:",inline" bson:",inline" json:",inline" mapstructure:",squash"` // provides IsCWLObject
	ID             string                                                                `yaml:"id,omitempty" bson:"id,omitempty" json:"id,omitempty" mapstructure:"id,omitempty"`
	SecondaryFiles interface{}                                                           `yaml:"secondaryFiles,omitempty" bson:"secondaryFiles,omitempty" json:"secondaryFiles,omitempty" mapstructure:"secondaryFiles,omitempty"` // TODO string | Expression | array<string | Expression>
	Format         []string                                                              `yaml:"format,omitempty" bson:"format,omitempty" json:"format,omitempty" mapstructure:"format,omitempty"`
	Streamable     bool                                                                  `yaml:"streamable,omitempty" bson:"streamable,omitempty" json:"streamable,omitempty" mapstructure:"streamable,omitempty"`
	Type           []CWLType_Type                                                        `yaml:"type,omitempty" bson:"type,omitempty" json:"type,omitempty" mapstructure:"type,omitempty"` // []CommandInputParameterType  CWLType | CommandInputRecordSchema | CommandInputEnumSchema | CommandInputArraySchema | string | array<CWLType | CommandInputRecordSchema | CommandInputEnumSchema | CommandInputArraySchema | string>
	Label          string                                                                `yaml:"label,omitempty" bson:"label,omitempty" json:"label,omitempty" mapstructure:"label,omitempty"`
	Description    string                                                                `yaml:"description,omitempty" bson:"description,omitempty" json:"description,omitempty" mapstructure:"description,omitempty"`
	InputBinding   *CommandLineBinding                                                   `yaml:"inputBinding,omitempty" bson:"inputBinding,omitempty" json:"inputBinding,omitempty" mapstructure:"inputBinding,omitempty"`
	Default        CWLType                                                               `yaml:"default,omitempty" bson:"default,omitempty" json:"default,omitempty" mapstructure:"default,omitempty"`
}

// NewCommandInputParameter _
func NewCommandInputParameter(v interface{}, schemata []CWLType_Type, context *WorkflowContext) (input_parameter *CommandInputParameter, err error) {

	//fmt.Println("NewCommandInputParameter:\n")
	//spew.Dump(v)
	//os.Exit(1)

	v, err = MakeStringMap(v, context)
	if err != nil {
		return
	}

	switch v.(type) {

	case map[string]interface{}:
		vMap, ok := v.(map[string]interface{})

		if !ok {
			err = fmt.Errorf("casting problem (a)")
			return
		}

		defaultValue, ok := vMap["default"]
		if ok {
			//fmt.Println("FOUND default key")
			default_input, xerr := NewCWLType("", defaultValue, context) // TODO return Int or similar
			if xerr != nil {
				err = xerr
				return
			}
			//spew.Dump(default_input)
			vMap["default"] = default_input
		}

		typeValue, ok := vMap["type"]
		if ok {

			typeValue, err = NewCommandInputParameterTypeArray(typeValue, schemata, context)
			if err != nil {
				err = fmt.Errorf("(NewCommandInputParameter) NewCommandInputParameterTypeArray returns: %s", err.Error())
				return
			}

			vMap["type"] = typeValue
		}

		formatValue, ok := vMap["format"]
		if ok {
			formatStr, isString := formatValue.(string)
			if isString {
				vMap["format"] = []string{formatStr}
			}

		}

		input_parameter = &CommandInputParameter{}
		err = mapstructure.Decode(v, input_parameter)
		if err != nil {
			err = fmt.Errorf("(NewCommandInputParameter) decode error: %s", err.Error())
			return
		}

	case string:
		//vString := v.(string)

		input_parameter = &CommandInputParameter{}

		var typeValue []CWLType_Type
		typeValue, err = NewCommandInputParameterTypeArray(v, schemata, context)
		if err != nil {
			err = fmt.Errorf("(NewCommandInputParameter) NewCommandInputParameterTypeArray returns: %s", err.Error())
			return
		}
		input_parameter.Type = typeValue

		return
	default:

		err = fmt.Errorf("(NewCommandInputParameter) type %s unknown", reflect.TypeOf(v))
		return
	}

	//input_parameter.InputBinding.Position = 1
	//input_parameter.InputBinding = nil
	//fmt.Println("NewCommandInputParameter:\n")
	//spew.Dump(input_parameter)

	//var data []byte
	//data, err = yaml.Marshal(input_parameter)
	//if err != nil {
	//	return
	//}
	//fmt.Println(string(data[:]))

	//os.Exit(1)

	return
}

// CreateCommandInputArray keyname will be converted into 'Id'-field
//   array<CommandInputParameter> | map<CommandInputParameter.id, CommandInputParameter.type> | map<CommandInputParameter.id, CommandInputParameter>
func CreateCommandInputArray(original interface{}, schemata []CWLType_Type, context *WorkflowContext) (err error, newArray []*CommandInputParameter) {

	original, err = MakeStringMap(original, context)
	if err != nil {
		err = fmt.Errorf("(NewInputParameter) MakeStringMap returned: %s", err.Error())
		return
	}

	//fmt.Println("CommandInputArray:")
	//spew.Dump(original)

	switch original.(type) {

	case map[string]interface{}:
		for k, v := range original.(map[string]interface{}) {

			//var input_parameter CommandInputParameter
			//mapstructure.Decode(v, &input_parameter)
			inputParameter, xerr := NewCommandInputParameter(v, schemata, context)
			if xerr != nil {
				err = fmt.Errorf("(CreateCommandInputArray) map[string]interface{} NewCommandInputParameter returned: %s", xerr.Error())
				return
			}

			inputParameter.ID = k

			//fmt.Printf("C")
			newArray = append(newArray, inputParameter)
			//fmt.Printf("D")

		}
	case []interface{}:
		for _, v := range original.([]interface{}) {

			inputParameter, xerr := NewCommandInputParameter(v, schemata, context)
			if xerr != nil {

				fmt.Println("CommandInputArray:")
				spew.Dump(original)

				err = fmt.Errorf("(CreateCommandInputArray) []interface{} NewCommandInputParameter returned: %s", xerr.Error())
				return
			}
			//var inputParameter CommandInputParameter
			//mapstructure.Decode(v, &inputParameter)

			//empty, err := NewEmpty(v)
			//if err != nil {
			//	return
			//}

			if inputParameter.ID == "" {
				err = fmt.Errorf("(CreateCommandInputArray) no id found")
				return
			}

			//fmt.Printf("C")
			newArray = append(newArray, inputParameter)
			//fmt.Printf("D")

		}
	case []*CommandInputParameter:

		newArray = original.([]*CommandInputParameter)

	default:
		err = fmt.Errorf("(CreateCommandInputArray) type unknown (%s)", reflect.TypeOf(original))
	}
	//spew.Dump(newArray)
	return
}
