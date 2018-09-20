package cwl

import (
	"fmt"
	//"github.com/davecgh/go-spew/spew"
	//"reflect"
)

// http://www.commonwl.org/v1.0/CommandLineTool.html#CommandInputRecordSchema
type CommandInputRecordSchema struct {
	RecordSchema `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // provides Type, Label, Name
	Fields       []CommandInputRecordField                                             `yaml:"fields,omitempty" json:"fields,omitempty" bson:"fields,omitempty"`
}

func NewCommandInputRecordSchema(native_map map[string]interface{}) (cirs *CommandInputRecordSchema, err error) {

	cirs = &CommandInputRecordSchema{}
	var rs *RecordSchema
	rs, err = NewRecordSchema(native_map)
	if err != nil {
		return
	}

	cirs.RecordSchema = *rs

	return
}

func NewCommandInputRecordSchemaFromInterface(native interface{}, schemata []CWLType_Type, context *WorkflowContext) (cirs *CommandInputRecordSchema, err error) {

	native, err = MakeStringMap(native, context)
	if err != nil {
		return
	}

	switch native.(type) {
	case map[string]interface{}:
		native_map, ok := native.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewInputRecordSchemaFromInterface) type switch error")
			return
		}

		cirs, err = NewCommandInputRecordSchema(native_map)
		if err != nil {
			return
		}

		fields, has_fields := native_map["fields"]
		if !has_fields {
			err = fmt.Errorf("(NewInputRecordSchemaFromInterface) no fields")
			return
		}

		var fields_array []interface{}
		fields_array, ok = fields.([]interface{})
		if !ok {
			err = fmt.Errorf("(NewInputRecordSchemaFromInterface) fields is not array")
			return
		}

		cirs.Fields, err = CreateCommandInputRecordFieldArray(fields_array, schemata, context)
		if err != nil {
			err = fmt.Errorf("(NewInputRecordSchemaFromInterface) CreateInputRecordFieldArray returns: %s", err.Error())
			return
		}
		return
	default:
		err = fmt.Errorf("(NewInputRecordSchemaFromInterface) error")

	}

	return

}
