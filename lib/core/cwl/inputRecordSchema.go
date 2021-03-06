package cwl

import (
	"fmt"
	//"github.com/davecgh/go-spew/spew"
	//"reflect"
)

// http://www.commonwl.org/v1.0/CommandLineTool.html#InputRecordSchema
type InputRecordSchema struct {
	RecordSchema `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // provides Type, Label, Name
	Fields       []InputRecordField                                                    `yaml:"fields,omitempty" json:"fields,omitempty" bson:"fields,omitempty"`
}

func NewInputRecordSchema(irs_map map[string]interface{}) (irs *InputRecordSchema, err error) {

	var ir *RecordSchema
	ir, err = NewRecordSchema(irs_map)
	if err != nil {
		return
	}

	irs = &InputRecordSchema{}
	irs.RecordSchema = *ir

	return
}

func NewInputRecordSchemaFromInterface(native interface{}, schemata []CWLType_Type, context *WorkflowContext) (irs *InputRecordSchema, err error) {
	//fmt.Println("native:")
	//spew.Dump(native)
	native, err = MakeStringMap(native, context)
	if err != nil {
		return
	}

	switch native.(type) {
	case map[string]interface{}:
		nativeMap, ok := native.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewInputRecordSchemaFromInterface) type switch error")
			return
		}

		irs, err = NewInputRecordSchema(nativeMap)
		if err != nil {
			err = fmt.Errorf("(NewInputRecordSchemaFromInterface) NewInputRecordSchema returns: %s", err.Error())
			return
		}

		fields, has_fields := nativeMap["fields"]
		if !has_fields {
			err = fmt.Errorf("(NewInputRecordSchemaFromInterface) no fields")
			return
		}

		irs.Fields, err = CreateInputRecordFieldArray(fields, schemata, context)
		if err != nil {
			err = fmt.Errorf("(NewInputRecordSchemaFromInterface) CreateInputRecordFieldArray returns: %s", err.Error())
			return
		}

	default:
		err = fmt.Errorf("(NewInputRecordSchemaFromInterface) error")
	}

	return

}
