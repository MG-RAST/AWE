package cwl

import (
	"fmt"
	"reflect"
)

type RecordSchema struct {
	NamedSchema `bson:",inline" yaml:",inline" json:",inline" mapstructure:",squash"` // provides Type, Label, Name
}

func (r *RecordSchema) GetID() string { return r.Name }
func (r *RecordSchema) Is_Type()      {}

func (r *RecordSchema) Type2String() string { return string(CWL_record) }

func NewRecordSchema(nativeMap map[string]interface{}) (rs *RecordSchema, err error) {

	rs = &RecordSchema{}

	label, has_label := nativeMap["label"]
	if has_label {
		var ok bool
		rs.Label, ok = label.(string)
		if !ok {
			err = fmt.Errorf("(NewInputRecordSchemaFromInterface) type error for label (%s)", reflect.TypeOf(label))
			return
		}
	}

	name, has_name := nativeMap["name"]
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
