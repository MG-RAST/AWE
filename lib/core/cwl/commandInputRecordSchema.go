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

// NewCommandInputRecordSchema _
func NewCommandInputRecordSchema(nativeMap map[string]interface{}) (cirs *CommandInputRecordSchema, err error) {

	cirs = &CommandInputRecordSchema{}
	var rs *RecordSchema
	rs, err = NewRecordSchema(nativeMap)
	if err != nil {
		return
	}

	cirs.RecordSchema = *rs

	return
}

// NewCommandInputRecordSchemaFromInterface _
func NewCommandInputRecordSchemaFromInterface(native interface{}, schemata []CWLType_Type, context *WorkflowContext) (cirs *CommandInputRecordSchema, err error) {

	native, err = MakeStringMap(native, context)
	if err != nil {
		return
	}

	//fmt.Println("(NewCommandInputRecordSchemaFromInterface): ")
	//spew.Dump(native)

	switch native.(type) {
	case map[string]interface{}:
		nativeMap, ok := native.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewCommandInputRecordSchemaFromInterface) type switch error")
			return
		}

		cirs, err = NewCommandInputRecordSchema(nativeMap)
		if err != nil {
			err = fmt.Errorf("(NewCommandInputRecordSchemaFromInterface) NewCommandInputRecordSchema returned: %s", err.Error())
			return
		}

		fields, hasFields := nativeMap["fields"]
		if !hasFields {
			err = fmt.Errorf("(NewCommandInputRecordSchemaFromInterface) no fields")
			return
		}

		// var fields_array []interface{}
		// fields_array, ok = fields.([]interface{})
		// if !ok {
		// 	err = fmt.Errorf("(NewCommandInputRecordSchemaFromInterface) fields is not array")
		// 	return
		// }

		cirs.Fields, err = CreateCommandInputRecordFieldArray(fields, schemata, context)
		if err != nil {
			err = fmt.Errorf("(NewCommandInputRecordSchemaFromInterface) CreateCommandInputRecordFieldArray returned: %s", err.Error())
			return
		}
		return
	default:
		err = fmt.Errorf("(NewCommandInputRecordSchemaFromInterface) error")

	}

	return

}
