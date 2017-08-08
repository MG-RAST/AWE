package cwl

import (
	"fmt"
	cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	"github.com/mitchellh/mapstructure"
)

type CommandOutputParameter struct {
	Id             string                       `yaml:"id,omitempty" bson:"id,omitempty" json:"id,omitempty"`
	SecondaryFiles []cwl_types.Expression       `yaml:"secondaryFiles,omitempty" bson:"secondaryFiles,omitempty" json:"secondaryFiles,omitempty"` // TODO string | Expression | array<string | Expression>
	Format         string                       `yaml:"format,omitempty" bson:"format,omitempty" json:"format,omitempty"`
	Streamable     bool                         `yaml:"streamable,omitempty" bson:"streamable,omitempty" json:"streamable,omitempty"`
	Type           []CommandOutputParameterType `yaml:"type,omitempty" bson:"type,omitempty" json:"type,omitempty"` // TODO CWLType | CommandInputRecordSchema | CommandInputEnumSchema | CommandInputArraySchema | string | array<CWLType | CommandInputRecordSchema | CommandInputEnumSchema | CommandInputArraySchema | string>
	Label          string                       `yaml:"label,omitempty" bson:"label,omitempty" json:"label,omitempty"`
	Description    string                       `yaml:"description,omitempty" bson:"description,omitempty" json:"description,omitempty"`
	OutputBinding  CommandOutputBinding         `yaml:"outputBinding,omitempty" bson:"outputBinding,omitempty" json:"outputBinding,omitempty"`
}

func NewCommandOutputParameter(original interface{}) (output_parameter *CommandOutputParameter, err error) {
	switch original.(type) {
	case map[interface{}]interface{}:

		original_map := original.(map[interface{}]interface{})

		outputBinding, ok := original_map["outputBinding"]
		if ok {
			original_map["outputBinding"], err = NewCommandOutputBinding(outputBinding)
			if err != nil {
				return
			}
		}

		COPtype, ok := original_map["type"]
		if ok {
			original_map["type"], err = NewCommandOutputParameterTypeArray(COPtype)
			if err != nil {
				return
			}
		}

		output_parameter = &CommandOutputParameter{}
		err = mapstructure.Decode(original, output_parameter)
		if err != nil {
			err = fmt.Errorf("(NewCommandOutputParameter) %s", err.Error())
			return
		}
	default:
		//spew.Dump(original)
		err = fmt.Errorf("NewCommandOutputParameter, unknown type")
	}
	//spew.Dump(new_array)
	return
}

func NewCommandOutputParameterArray(original interface{}) (copa *[]CommandOutputParameter, err error) {

	switch original.(type) {
	case map[interface{}]interface{}:
		cop, xerr := NewCommandOutputParameter(original)
		if xerr != nil {
			err = xerr
			return
		}
		copa = &[]CommandOutputParameter{*cop}
	case []interface{}:
		copa_nptr := []CommandOutputParameter{}

		original_array := original.([]interface{})

		for _, element := range original_array {
			cop, xerr := NewCommandOutputParameter(element)
			if xerr != nil {
				err = xerr
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
