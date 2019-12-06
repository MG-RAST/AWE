package cwl

import (
	"fmt"
	"reflect"
)

type OutputArraySchema struct { // Items, Type , Label
	ArraySchema   `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
	OutputBinding *CommandOutputBinding `yaml:"outputBinding,omitempty" bson:"outputBinding,omitempty" json:"outputBinding,omitempty"`
}

//func (c *CommandOutputArraySchema) Is_CommandOutputParameterType() {}

func (c OutputArraySchema) Type2String() string { return "OutputArraySchema" }
func (c OutputArraySchema) GetID() string       { return "" }
func (c OutputArraySchema) Is_Type()            {}

func NewOutputArraySchema() (coas *OutputArraySchema) {

	coas = &OutputArraySchema{}
	coas.Type = CWLArray

	return
}

func NewOutputArraySchemaFromInterface(original interface{}, schemata []CWLType_Type, context *WorkflowContext) (coas *OutputArraySchema, err error) {

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	switch original.(type) {

	case map[string]interface{}:

		original_map, ok := original.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewOutputArraySchemaFromInterface) type error b")
			return
		}
		var as *ArraySchema
		as, err = NewArraySchemaFromMap(original_map, schemata, "Output", context)
		if err != nil {
			err = fmt.Errorf("(NewOutputArraySchemaFromInterface) NewArraySchemaFromInterface returned: %s", err.Error())
			return
		}

		coas = &OutputArraySchema{}
		coas.ArraySchema = *as

		outputBinding, has_outputBinding := original_map["outputBinding"]
		if has_outputBinding {

			coas.OutputBinding, err = NewCommandOutputBinding(outputBinding, context)
			if err != nil {
				err = fmt.Errorf("(NewOutputArraySchemaFromInterface) NewCommandOutputBinding returned: %s", err.Error())
				return
			}
		}

		//err = mapstructure.Decode(original, coas)
		//if err != nil {
		//	err = fmt.Errorf("(NewOutputArraySchema) %s", err.Error())
		//	return
		//}
	default:
		err = fmt.Errorf("NewOutputArraySchemaFromInterface, unknown type %s", reflect.TypeOf(original))
	}
	return
}
