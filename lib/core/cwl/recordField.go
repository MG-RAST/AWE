package cwl

import (
	"fmt"
	"reflect"
)

// http://www.commonwl.org/v1.0/CommandLineTool.html#InputRecordField
type RecordField struct {
	Name string         `yaml:"name,omitempty" json:"name,omitempty" bson:"name,omitempty"`
	Type []CWLType_Type `yaml:"type,omitempty" json:"type,omitempty" bson:"type,omitempty"` // CWLType | InputRecordSchema | InputEnumSchema | InputArraySchema | string | array<CWLType | InputRecordSchema | InputEnumSchema | InputArraySchema | string>
	Doc  string         `yaml:"doc,omitempty" json:"doc,omitempty" bson:"doc,omitempty"`

	Label string `yaml:"label,omitempty" json:"label,omitempty" bson:"label,omitempty"`
}

// used for :
// InputRecordField
//     CWLType | InputRecordSchema | InputEnumSchema | InputArraySchema | string | array<CWLType | InputRecordSchema | InputEnumSchema | InputArraySchema | string>
// OutputRecordField
//     CWLType | OutputRecordSchema | OutputEnumSchema | OutputArraySchema | string | array<CWLType | OutputRecordSchema | OutputEnumSchema | OutputArraySchema | string>

// CommandInputRecordField
//     CWLType | CommandInputRecordSchema | CommandInputEnumSchema | CommandInputArraySchema | string | array<CWLType | CommandInputRecordSchema | CommandInputEnumSchema | CommandInputArraySchema | string>
// CommandOutputRecordField
//     CWLType | CommandOutputRecordSchema | CommandOutputEnumSchema | CommandOutputArraySchema | string | array<CWLType | CommandOutputRecordSchema | CommandOutputEnumSchema | CommandOutputArraySchema | string>

func NewRecordFieldFromInterface(native interface{}, schemata []CWLType_Type, context_p string, context *WorkflowContext) (rf *RecordField, err error) {

	native, err = MakeStringMap(native, context)
	if err != nil {
		return
	}

	switch native.(type) {
	case map[string]interface{}:
		nativeMap, ok := native.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewRecordFieldFromInterface) type switch error")
			return
		}

		rf = &RecordField{}

		name, has_name := nativeMap["name"]
		if has_name {
			var ok bool
			rf.Name, ok = name.(string)
			if !ok {
				err = fmt.Errorf("(NewRecordFieldFromInterface) type error for name (%s)", reflect.TypeOf(name))
				return
			}
		}

		label, has_label := nativeMap["label"]
		if has_label {
			var ok bool
			rf.Label, ok = label.(string)
			if !ok {
				err = fmt.Errorf("(NewRecordFieldFromInterface) type error for label (%s)", reflect.TypeOf(label))
				return
			}
		}

		doc, has_doc := nativeMap["doc"]
		if has_doc {
			var ok bool
			rf.Doc, ok = doc.(string)
			if !ok {
				err = fmt.Errorf("(NewRecordFieldFromInterface) type error for doc (%s)", reflect.TypeOf(doc))
				return
			}
		}

		the_type, has_type := nativeMap["type"]
		if has_type {

			rf.Type, err = NewCWLType_TypeArray(the_type, schemata, context_p, false, context)
			if err != nil {
				err = fmt.Errorf("(NewRecordFieldFromInterface) NewCWLTypeArray returned: %s", err.Error())
				return
			}

		}

		return

	default:
		err = fmt.Errorf("(NewRecordFieldFromInterface) unknown type")
	}

	return
}
