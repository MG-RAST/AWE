package cwl

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
)

// http://www.commonwl.org/v1.0/CommandLineTool.html#InputRecordSchema
type InputRecordSchema struct {
	Type   string
	Fields []InputRecordField `yaml:"fields,omitempty" json:"fields,omitempty" bson:"fields,omitempty"`
	Label  string             `yaml:"label,omitempty" json:"label,omitempty" bson:"label,omitempty"`
	Id     string             `yaml:"-" json:"-" bson:"id,omitempty"`
}

//func (r *InputRecordSchema) GetClass() string { return "record" }
func (r *InputRecordSchema) GetId() string { return r.Id }

//func (r *InputRecordSchema) Is_CWL_minimal()     {}
func (r *InputRecordSchema) Is_Type() {}

func (r *InputRecordSchema) Type2String() string { return "record" }

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

		irs = &InputRecordSchema{}

		the_type, has_type := native_map["type"]
		if has_type {
			var ok bool
			irs.Type, ok = the_type.(string)
			if !ok {
				err = fmt.Errorf("(NewInputRecordSchemaFromInterface) type error")
				return
			}
		}

		label, has_label := native_map["label"]
		if has_label {
			var ok bool
			irs.Label, ok = label.(string)
			if !ok {
				err = fmt.Errorf("(NewInputRecordSchemaFromInterface) type error for label")
				return
			}
		}

		id, has_id := native_map["id"]
		if has_id {
			var ok bool
			irs.Id, ok = id.(string)
			if !ok {
				err = fmt.Errorf("(NewInputRecordSchemaFromInterface) type error for id")
				return
			}
		}
	default:
		err = fmt.Errorf("(NewInputRecordSchemaFromInterface) error")
		return
	}

	spew.Dump(native)

	panic("blubb")

	return

}
