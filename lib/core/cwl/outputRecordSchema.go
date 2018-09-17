package cwl

import (
	"fmt"
	//"github.com/davecgh/go-spew/spew"
	//"reflect"
)

// https://www.commonwl.org/v1.0/Workflow.html#OutputRecordSchema

type OutputRecordSchema struct {
	RecordSchema `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // provides Type, Label, Name
	Fields       []OutputRecordField                                                   `yaml:"fields,omitempty" json:"fields,omitempty" bson:"fields,omitempty"`
}

func NewOutputRecordSchema(irs_map map[string]interface{}) (ors *OutputRecordSchema, err error) {

	var rs *RecordSchema
	rs, err = NewRecordSchema(irs_map)
	if err != nil {
		return
	}

	ors = &OutputRecordSchema{}
	ors.RecordSchema = *rs

	return
}

func NewOutputRecordSchemaFromInterface(native interface{}, schemata []CWLType_Type, context *WorkflowContext) (ors *OutputRecordSchema, err error) {

	native, err = MakeStringMap(native, context)
	if err != nil {
		return
	}

	switch native.(type) {
	case map[string]interface{}:
		native_map, ok := native.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewOutputRecordSchemaFromInterface) type switch error")
			return
		}

		ors, err = NewOutputRecordSchema(native_map)
		if err != nil {
			err = fmt.Errorf("(NewOutputRecordSchemaFromInterface) NewOutputRecordSchema returns: %s", err.Error())
			return
		}

		fields, has_fields := native_map["fields"]
		if !has_fields {
			err = fmt.Errorf("(NewOutputRecordSchemaFromInterface) no fields")
			return
		}

		var fields_array []interface{}
		fields_array, ok = fields.([]interface{})
		if !ok {
			err = fmt.Errorf("(NewOutputRecordSchemaFromInterface) fields is not array")
			return
		}

		ors.Fields, err = CreateOutputRecordFieldArray(fields_array, schemata, context)
		if err != nil {
			err = fmt.Errorf("(NewOutputRecordSchemaFromInterface) CreateOutputRecordFieldArray returns: %s", err.Error())
			return
		}
		return
	default:
		err = fmt.Errorf("(NewOutputRecordSchemaFromInterface) error")
	}

	return

}
