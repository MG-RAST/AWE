package cwl

import (
	"fmt"
	//"github.com/davecgh/go-spew/spew"
	"reflect"
)

// http://www.commonwl.org/v1.0/CommandLineTool.html#InputRecordSchema
type InputRecordSchema struct {
	Type   CWLType_Type       `yaml:"type,omitempty" json:"type,omitempty" bson:"type,omitempty"`
	Fields []InputRecordField `yaml:"fields,omitempty" json:"fields,omitempty" bson:"fields,omitempty"`
	Label  string             `yaml:"label,omitempty" json:"label,omitempty" bson:"label,omitempty"`
	Name   string             `yaml:"name,omitempty" json:"name,omitempty" bson:"name,omitempty"`
	//Id     string             `yaml:"-" json:"-" bson:"id,omitempty"`
}

//func (r *InputRecordSchema) GetClass() string { return "record" }
func (r *InputRecordSchema) GetId() string { return r.Name }

//func (r *InputRecordSchema) Is_CWL_minimal()     {}
func (r *InputRecordSchema) Is_Type() {}

func (r *InputRecordSchema) Type2String() string { return "record" }

func NewInputRecordSchema() *InputRecordSchema {
	return &InputRecordSchema{Type: CWL_record}
}

func NewInputRecordSchemaFromInterface(native interface{}, schemata []CWLType_Type) (irs *InputRecordSchema, err error) {

	native, err = MakeStringMap(native)
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

		irs = NewInputRecordSchema()

		//the_type, has_type := native_map["type"]
		//if has_type {
		//	var ok bool
		//	irs.Type, ok = the_type.(string)
		//	if !ok {
		//		err = fmt.Errorf("(NewInputRecordSchemaFromInterface) type error ()%s)", reflect.TypeOf(the_type))
		//		return
		//	}
		//}

		label, has_label := native_map["label"]
		if has_label {
			var ok bool
			irs.Label, ok = label.(string)
			if !ok {
				err = fmt.Errorf("(NewInputRecordSchemaFromInterface) type error for label (%s)", reflect.TypeOf(label))
				return
			}
		}

		name, has_name := native_map["name"]
		if has_name {
			var ok bool
			irs.Name, ok = name.(string)
			if !ok {
				err = fmt.Errorf("(NewInputRecordSchemaFromInterface) type error for name (%s)", reflect.TypeOf(name))
				return
			}
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

		irs.Fields, err = CreateInputRecordFieldArray(fields_array, schemata)
		if err != nil {
			err = fmt.Errorf("(NewInputRecordSchemaFromInterface) CreateInputRecordFieldArray returns: %s", err.Error())
			return
		}
		return
	default:
		err = fmt.Errorf("(NewInputRecordSchemaFromInterface) error")
		return
	}

	return

}
