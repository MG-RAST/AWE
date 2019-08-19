package cwl

import (
	"fmt"
	"reflect"

	"github.com/davecgh/go-spew/spew"
	//"github.com/davecgh/go-spew/spew"
	//"reflect"
)

// http://www.commonwl.org/v1.0/CommandLineTool.html#CommandInputRecordField
type CommandInputRecordField struct {
	RecordField  `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // name type, doc , label
	InputBinding *CommandLineBinding                                                   `yaml:"inputBinding,omitempty" json:"inputBinding,omitempty" bson:"inputBinding,omitempty"`
}

// NewCommandInputRecordField _
func NewCommandInputRecordField(native interface{}, schemata []CWLType_Type, context *WorkflowContext) (crf *CommandInputRecordField, err error) {

	native, err = MakeStringMap(native, context)
	if err != nil {
		return
	}

	switch native.(type) {
	case map[interface{}]interface{}:
		err = fmt.Errorf("(NewCommandInputRecordField) need map[string]interface{} but got map[interface{}]interface{}")
		return

	case map[string]interface{}:
		var rf *RecordField
		rf, err = NewRecordFieldFromInterface(native, "", schemata, "CommandInput", context)
		if err != nil {
			err = fmt.Errorf("(NewCommandInputRecordField) NewRecordFieldFromInterface returned: %s", err.Error())
			return
		}

		crf = &CommandInputRecordField{}
		crf.RecordField = *rf

		nativeMap, ok := native.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewCommandInputRecordField) type assertion error (%s)", reflect.TypeOf(native))
			return
		}

		inputBinding, hasInputBinding := nativeMap["inputBinding"]
		if hasInputBinding {

			crf.InputBinding, err = NewCommandLineBinding(inputBinding, context)
			if err != nil {
				err = fmt.Errorf("(NewCommandInputRecordField) NewCWLTypeArray returned: %s", err.Error())
				return
			}
		}
	case []interface{}:
		// elememts probably are only types..

		var rf *RecordField
		rf, err = NewRecordFieldFromInterface(native, "", schemata, "CommandInput", context)
		if err != nil {
			err = fmt.Errorf("(NewCommandInputRecordField) NewRecordFieldFromInterface returned: %s", err.Error())
			return
		}

		crf = &CommandInputRecordField{}
		crf.RecordField = *rf

	default:
		fmt.Println("NewCommandInputRecordField:")
		spew.Dump(native)
		err = fmt.Errorf("(NewCommandInputRecordField) type not supported: %s", reflect.TypeOf(native))
	}

	return
}

// CreateCommandInputRecordFieldArray _
func CreateCommandInputRecordFieldArray(native interface{}, schemata []CWLType_Type, context *WorkflowContext) (irfa []CommandInputRecordField, err error) {

	native, err = MakeStringMap(native, context)
	if err != nil {
		return
	}

	//fmt.Println("CreateCommandInputRecordFieldArray:")
	//spew.Dump(native)

	switch native.(type) {
	case []interface{}:
		nativeArray := native.([]interface{})
		for _, elem := range nativeArray {

			var irf *CommandInputRecordField
			irf, err = NewCommandInputRecordField(elem, schemata, context)
			if err != nil {
				err = fmt.Errorf("(CreateCommandInputRecordFieldArray) returned: %s", err.Error())
				return

			}

			irfa = append(irfa, *irf)
		}

	case map[string]interface{}:
		nativeMap := native.(map[string]interface{})
		for id, elem := range nativeMap {

			var irf *CommandInputRecordField
			irf, err = NewCommandInputRecordField(elem, schemata, context)
			if err != nil {
				err = fmt.Errorf("(CreateCommandInputRecordFieldArray) returned: %s", err.Error())
				return

			}
			irf.Name = id
			irfa = append(irfa, *irf)
		}

	default:
		err = fmt.Errorf("(CreateCommandInputRecordFieldArray) type not supported: %s", reflect.TypeOf(native))
		return
	}

	return
}
