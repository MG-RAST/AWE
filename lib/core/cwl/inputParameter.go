package cwl

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"

	"reflect"
	//"strings"
)

// InputParameter https://www.commonwl.org/v1.0/Workflow.html#InputParameter
type InputParameter struct {
	CWLObjectImpl  `yaml:",inline" bson:",inline" json:",inline" mapstructure:",squash"` // provides IsCWLObject
	IdentifierImpl `yaml:",inline" bson:",inline" json:",inline" mapstructure:",squash"` // provides id
	Label          string                                                                `yaml:"label,omitempty" bson:"label,omitempty" json:"label,omitempty" mapstructure:"label,omitempty"`
	SecondaryFiles interface{}                                                           `yaml:"secondaryFiles,omitempty" bson:"secondaryFiles,omitempty" json:"secondaryFiles,omitempty" mapstructure:"secondaryFiles,omitempty"` // TODO string | Expression | array<string | Expression>
	Format         []string                                                              `yaml:"format,omitempty" bson:"format,omitempty" json:"format,omitempty" mapstructure:"format,omitempty"`
	Streamable     bool                                                                  `yaml:"streamable,omitempty" bson:"streamable,omitempty" json:"streamable,omitempty" mapstructure:"streamable,omitempty"`
	Doc            string                                                                `yaml:"doc,omitempty" bson:"doc,omitempty" json:"doc,omitempty" mapstructure:"doc,omitempty"`
	InputBinding   *CommandLineBinding                                                   `yaml:"inputBinding,omitempty" bson:"inputBinding,omitempty" json:"inputBinding,omitempty" mapstructure:"inputBinding,omitempty"` //TODO
	Default        CWLType                                                               `yaml:"default,omitempty" bson:"default,omitempty" json:"default,omitempty" mapstructure:"default,omitempty"`
	Type           interface{}                                                           `yaml:"type,omitempty" bson:"type,omitempty" json:"type,omitempty" mapstructure:"type,omitempty"` // TODO CWLType | InputRecordSchema | InputEnumSchema | InputArraySchema | string | array<CWLType | InputRecordSchema | InputEnumSchema | InputArraySchema | string>
}

// GetClass _
func (i InputParameter) GetClass() string { return "InputParameter" }

//func (i InputParameter) GetID() string    { return i.ID }
//func (i InputParameter) SetID(id string)  { i.ID = id }
// IsCWLMinimal _
func (i InputParameter) IsCWLMinimal() {}

// NewInputParameter _
func NewInputParameter(original interface{}, schemata []CWLType_Type, context *WorkflowContext) (inputParameter *InputParameter, err error) {

	//fmt.Println("---- NewInputParameter ----")
	//spew.Dump(original)

	original, err = MakeStringMap(original, context)
	if err != nil {
		err = fmt.Errorf("(NewInputParameter) MakeStringMap returned: %s", err.Error())
		return
	}
	//spew.Dump(original)

	inputParameter = &InputParameter{}

	switch original.(type) {
	case string:

		typeString := original.(string)

		var originalTypeArray []CWLType_Type
		originalTypeArray, err = NewCWLType_TypeFromString(schemata, typeString, "Input", context)
		if err != nil {
			err = fmt.Errorf("(NewInputParameter) NewCWLType_TypeFromString returned: %s", err.Error())
			return
		}

		//inputParameter_type, xerr := NewInputParameterType(type_string_lower)
		//if xerr != nil {
		//	err = xerr
		//	return
		//}

		inputParameter.Type = originalTypeArray

		//case int:
		//inputParameter_type, xerr := NewInputParameterTypeArray("int")
		//if xerr != nil {
		//	err = xerr
		//	return
		//}

		//inputParameter.Type = inputParameter_type
	case []interface{}:

		var originalTypeArray []CWLType_Type
		originalTypeArray, err = NewCWLType_TypeArray(original, schemata, "Input", true, context)
		if err != nil {
			err = fmt.Errorf("(NewInputParameter) NewCWLType_Type returned: %s", err.Error())
			return
		}

		inputParameter.Type = originalTypeArray

	case map[string]interface{}:

		originalMap := original.(map[string]interface{})

		inputParameterDefault, ok := originalMap["default"]
		if ok {
			originalMap["default"], err = NewCWLType("", "", inputParameterDefault, context)
			if err != nil {
				err = fmt.Errorf("(NewInputParameter) NewCWLType returned: %s", err.Error())
				return
			}
		}

		inputParameterTypeIf, ok := originalMap["type"]
		if ok {
			switch inputParameterTypeIf.(type) {
			case []interface{}:
				var inputParameterTypeArray []CWLType_Type
				inputParameterTypeArray, err = NewCWLType_TypeArray(inputParameterTypeIf, schemata, "Input", false, context)
				if err != nil {
					fmt.Println("inputParameterTypeIf:")
					spew.Dump(inputParameterTypeIf)
					err = fmt.Errorf("(NewInputParameter) NewCWLType_TypeArray returns: %s", err.Error())
					return
				}
				if len(inputParameterTypeArray) == 0 {
					err = fmt.Errorf("(NewInputParameter) len(inputParameterTypeArray) == 0")
					return
				}
				originalMap["type"] = inputParameterTypeArray
			default:
				var inputParameterTypeArray []CWLType_Type
				var inputParameterType CWLType_Type
				inputParameterTypeArray, err = NewCWLType_Type(schemata, inputParameterTypeIf, "Input", context)
				if err != nil {
					fmt.Println("inputParameterType:")
					spew.Dump(inputParameterType)
					err = fmt.Errorf("(NewInputParameter) NewCWLType_Type returns: %s", err.Error())
					return
				}
				if len(inputParameterTypeArray) == 1 {
					inputParameterType = inputParameterTypeArray[0]
					originalMap["type"] = inputParameterType
				} else {
					originalMap["type"] = inputParameterTypeArray
				}

			}
		}

		err = mapstructure.Decode(original, inputParameter)
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

	return
}

// NewInputParameterArray _
func NewInputParameterArray(original interface{}, schemata []CWLType_Type, context *WorkflowContext) (new_array []InputParameter, err error) {

	original, err = MakeStringMap(original, context)
	if err != nil {
		err = fmt.Errorf("(NewInputParameterArray) MakeStringMap returned: %s", err.Error())
		return
	}

	//fmt.Println("NewInputParameterArray:")
	//spew.Dump(original)

	switch original.(type) {
	case map[string]interface{}:
		originalMap := original.(map[string]interface{})
		for id, v := range originalMap {
			//fmt.Printf("A")

			inputParameter, xerr := NewInputParameter(v, schemata, context)
			if xerr != nil {
				err = fmt.Errorf("(NewInputParameterArray) A NewInputParameter returned: %s", xerr.Error())
				return
			}

			inputParameter.ID = id

			if inputParameter.ID == "" {
				err = fmt.Errorf("(NewInputParameterArray) ID is missing")
				return
			}

			//fmt.Printf("C")
			new_array = append(new_array, *inputParameter)
			//fmt.Printf("D")

		}
	case []interface{}:
		originalArray := original.([]interface{})
		for _, v := range originalArray {
			//fmt.Printf("A")

			inputParameter, xerr := NewInputParameter(v, schemata, context)
			if xerr != nil {
				err = fmt.Errorf("(NewInputParameterArray) B NewInputParameter returned: %s", xerr.Error())
				return
			}

			if inputParameter.ID == "" {
				err = fmt.Errorf("(NewInputParameterArray) ID is missing")
				return
			}

			//fmt.Printf("C")
			new_array = append(new_array, *inputParameter)
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

// GetTypes convenience function taht always retuns an array
func (ip *InputParameter) GetTypes() (result []CWLType_Type, err error) {

	ipType := ip.Type

	switch ipType.(type) {
	case []interface{}:

		workflowInputParameterTypesArrayIf := ipType.([]interface{})
		for _, tIf := range workflowInputParameterTypesArrayIf {
			t, ok := tIf.(CWLType_Type)
			if !ok {
				err = fmt.Errorf("(GetTypes) cannot convert type")
				return
			}
			result = append(result, t)
		}

	case []CWLType_Type:
		result = ipType.([]CWLType_Type)

	default:
		t, ok := ipType.(CWLType_Type)
		if !ok {

			err = fmt.Errorf("(GetTypes) cannot convert type (%s)", reflect.TypeOf(ipType))
			return
		}

		result = []CWLType_Type{t}
	}

	return
}
