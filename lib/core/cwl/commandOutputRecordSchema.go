package cwl

import (
	"fmt"
)

type CommandOutputRecordSchema struct {
	RecordSchema `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // provides Type, Label, Name
	//Type   string                     `yaml:"type,omitempty" bson:"type,omitempty" json:"type,omitempty" mapstructure:"type,omitempty"` // Must be record
	Fields []CommandOutputRecordField `yaml:"fields,omitempty" bson:"fields,omitempty" json:"fields,omitempty" mapstructure:"fields,omitempty"`
	//Label  string                     `yaml:"label,omitempty" bson:"label,omitempty" json:"label,omitempty" mapstructure:"label,omitempty"`
}

//func (c *CommandOutputRecordSchema) Is_CommandOutputParameterType() {}
func (c *CommandOutputRecordSchema) Is_Type()            {}
func (c *CommandOutputRecordSchema) Type2String() string { return "CommandOutputRecordSchema" }
func (c *CommandOutputRecordSchema) GetId() string       { return "" }

func NewCommandOutputRecordSchema() (schema *CommandOutputRecordSchema, err error) {

	schema = &CommandOutputRecordSchema{}
	schema.RecordSchema = RecordSchema{}
	// err = mapstructure.Decode(v, schema)
	// if err != nil {
	// 	err = fmt.Errorf("(NewCommandOutputRecordSchema) decode error: %s", err.Error())
	// 	return
	// }

	return
}

func NewCommandOutputRecordSchemaFromInterface(native interface{}, schemata []CWLType_Type) (cirs *CommandOutputRecordSchema, err error) {

	native, err = MakeStringMap(native)
	if err != nil {
		return
	}

	switch native.(type) {
	case map[string]interface{}:
		native_map, ok := native.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewCommandoutputRecordSchemaFromInterface) type switch error")
			return
		}

		var rs *RecordSchema
		rs, err = NewRecordSchema(native_map)
		if err != nil {
			return
		}

		cirs, err = NewCommandOutputRecordSchema()
		if err != nil {
			err = fmt.Errorf("(NewCommandoutputRecordSchemaFromInterface) NewCommandOutputRecordSchema returns: %s", err.Error())
			return
		}

		fields, has_fields := native_map["fields"]
		if !has_fields {
			err = fmt.Errorf("(NewCommandoutputRecordSchemaFromInterface) no fields")
			return
		}

		var fields_array []interface{}
		fields_array, ok = fields.([]interface{})
		if !ok {
			err = fmt.Errorf("(NewCommandoutputRecordSchemaFromInterface) fields is not array")
			return
		}

		cirs.Fields, err = CreateCommandOutputRecordFieldArray(fields_array, schemata)
		if err != nil {
			err = fmt.Errorf("(NewCommandoutputRecordSchemaFromInterface) CreateInputRecordFieldArray returns: %s", err.Error())
			return
		}

		cirs.RecordSchema = *rs

		return
	default:
		err = fmt.Errorf("(NewCommandoutputRecordSchemaFromInterface) error")

	}

	return

}
