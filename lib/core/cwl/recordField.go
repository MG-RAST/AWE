package cwl

import (
	"fmt"
	"reflect"

	"github.com/davecgh/go-spew/spew"
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

// NewRecordFieldFromInterface _
func NewRecordFieldFromInterface(native interface{}, name string, schemata []CWLType_Type, context_p string, context *WorkflowContext) (rf *RecordField, err error) {

	native, err = MakeStringMap(native, context)
	if err != nil {
		return
	}

	rf = &RecordField{}

	if name != "" {
		rf.Name = name
	}

	switch native.(type) {
	case map[string]interface{}:
		nativeMap, ok := native.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewRecordFieldFromInterface) type switch error")
			return
		}

		nameIf, hasName := nativeMap["name"]
		if hasName {
			var ok bool
			var nameStr string
			nameStr, ok = nameIf.(string)

			if !ok {
				err = fmt.Errorf("(NewRecordFieldFromInterface) type error for name (%s)", reflect.TypeOf(name))
				return
			}
			if nameStr != "" {
				rf.Name = nameStr
			}

		}

		label, hasLabel := nativeMap["label"]
		if hasLabel {
			var ok bool
			rf.Label, ok = label.(string)
			if !ok {
				err = fmt.Errorf("(NewRecordFieldFromInterface) type error for label (%s)", reflect.TypeOf(label))
				return
			}
		}

		doc, hasDoc := nativeMap["doc"]
		if hasDoc {
			var ok bool
			rf.Doc, ok = doc.(string)
			if !ok {
				err = fmt.Errorf("(NewRecordFieldFromInterface) type error for doc (%s)", reflect.TypeOf(doc))
				return
			}
		}

		typeIf, hasType := nativeMap["type"]
		if hasType {

			rf.Type, err = NewCWLType_TypeArray(typeIf, schemata, context_p, false, context)
			if err != nil {
				err = fmt.Errorf("(NewRecordFieldFromInterface) NewCWLTypeArray returned: %s", err.Error())
				return
			}

		}

		return
	case []interface{}:
		// this is only types
		rf.Type, err = NewCWLType_TypeArray(native, schemata, context_p, false, context)
		if err != nil {
			err = fmt.Errorf("(NewRecordFieldFromInterface) NewCWLTypeArray returned: %s", err.Error())
			return
		}

	default:
		spew.Dump(native)
		err = fmt.Errorf("(NewRecordFieldFromInterface) unknown type %s", reflect.TypeOf(native))
	}

	return
}
