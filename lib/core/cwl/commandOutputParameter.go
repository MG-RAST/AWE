package cwl

import (
	"fmt"
	"reflect"

	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
)

type CommandOutputParameter struct {
	OutputParameter `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // provides Id, Label, SecondaryFiles, Format, Streamable, OutputBinding, Type

	Description string `yaml:"description,omitempty" bson:"description,omitempty" json:"description,omitempty" mapstructure:"description,omitempty"`
}

func NewCommandOutputParameter(original interface{}, schemata []CWLType_Type) (output_parameter *CommandOutputParameter, err error) {

	original, err = MakeStringMap(original)
	if err != nil {
		return
	}

	switch original.(type) {

	case map[string]interface{}:

		original_map, ok := original.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewCommandOutputParameter) type error")
			return
		}

		err = NormalizeOutputParameter(original_map)
		if err != nil {
			err = fmt.Errorf("(NewCommandOutputParameter) NormalizeOutputParameter returns %s", err.Error())
			return
		}

		COPtype, ok := original_map["type"]
		if ok {
			original_map["type"], err = NewCommandOutputParameterTypeArray(COPtype, schemata)
			if err != nil {
				err = fmt.Errorf("(NewCommandOutputParameter) NewCommandOutputParameterTypeArray returns %s", err.Error())
				return
			}
		}

		output_parameter = &CommandOutputParameter{}
		err = mapstructure.Decode(original, output_parameter)
		if err != nil {
			err = fmt.Errorf("(NewCommandOutputParameter) mapstructure returned: %s", err.Error())
			return
		}
	default:
		spew.Dump(original)
		err = fmt.Errorf("NewCommandOutputParameter, unknown type %s", reflect.TypeOf(original))
	}
	//spew.Dump(new_array)
	return
}

func NewCommandOutputParameterArray(original interface{}, schemata []CWLType_Type) (copa *[]CommandOutputParameter, err error) {

	switch original.(type) {
	case map[interface{}]interface{}:
		cop, xerr := NewCommandOutputParameter(original, schemata)
		if xerr != nil {
			err = fmt.Errorf("(NewCommandOutputParameterArray) a NewCommandOutputParameter returns: %s", xerr.Error())
			return
		}
		copa = &[]CommandOutputParameter{*cop}
	case []interface{}:
		copa_nptr := []CommandOutputParameter{}

		original_array := original.([]interface{})

		for _, element := range original_array {
			cop, xerr := NewCommandOutputParameter(element, schemata)
			if xerr != nil {
				err = fmt.Errorf("(NewCommandOutputParameterArray) b NewCommandOutputParameter returns: %s", xerr.Error())
				return
			}
			copa_nptr = append(copa_nptr, *cop)
		}

		copa = &copa_nptr
	default:
		err = fmt.Errorf("NewCommandOutputParameterArray, unknown type")
	}
	return

}
