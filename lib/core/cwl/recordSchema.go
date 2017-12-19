package cwl

import (
	"fmt"
	"reflect"
)

type RecordSchema struct {
	Type  CWLType_Type `yaml:"type,omitempty" json:"type,omitempty" bson:"type,omitempty"`
	Label string       `yaml:"label,omitempty" json:"label,omitempty" bson:"label,omitempty"`
	Name  string       `yaml:"name,omitempty" json:"name,omitempty" bson:"name,omitempty"`
}

func (r *RecordSchema) GetId() string { return r.Name }
func (r *RecordSchema) Is_Type()      {}

func (r *RecordSchema) Type2String() string { return string(CWL_record) }

func NewRecordSchema(native_map map[string]interface{}) (rs *RecordSchema, err error) {

	rs = &RecordSchema{}

	label, has_label := native_map["label"]
	if has_label {
		var ok bool
		rs.Label, ok = label.(string)
		if !ok {
			err = fmt.Errorf("(NewInputRecordSchemaFromInterface) type error for label (%s)", reflect.TypeOf(label))
			return
		}
	}

	name, has_name := native_map["name"]
	if has_name {
		var ok bool
		rs.Name, ok = name.(string)
		if !ok {
			err = fmt.Errorf("(NewInputRecordSchemaFromInterface) type error for name (%s)", reflect.TypeOf(name))
			return
		}
	}

	rs.Type = CWL_record

	return
}
