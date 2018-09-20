package cwl

import (
	"fmt"
	"reflect"
	//"github.com/davecgh/go-spew/spew"
	//"reflect"
)

// http://www.commonwl.org/v1.0/CommandLineTool.html#InputRecordField
type CommandInputRecordField struct {
	RecordField  `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // name type, doc , label
	InputBinding *CommandLineBinding                                                   `yaml:"inputBinding,omitempty" json:"inputBinding,omitempty" bson:"inputBinding,omitempty"`
}

func NewCommandInputRecordField(native interface{}, schemata []CWLType_Type, context *WorkflowContext) (crf *CommandInputRecordField, err error) {

	native, err = MakeStringMap(native, context)
	if err != nil {
		return
	}

	switch native.(type) {
	case map[interface{}]interface{}:
		err = fmt.Errorf("(NewCommandInputRecordField) need map[string]interface{} but got map[interface{}]interface{}")
		return
	}

	var rf *RecordField
	rf, err = NewRecordFieldFromInterface(native, schemata, "CommandInput", context)
	if err != nil {
		err = fmt.Errorf("(NewCommandInputRecordField) NewRecordFieldFromInterface returned: %s", err.Error())
		return
	}

	crf = &CommandInputRecordField{}
	crf.RecordField = *rf

	native_map, ok := native.(map[string]interface{})
	if !ok {
		err = fmt.Errorf("(NewInputRecordFieldFromInterface) type assertion error (%s)", reflect.TypeOf(native))
		return
	}

	inputBinding, has_inputBinding := native_map["inputBinding"]
	if has_inputBinding {

		crf.InputBinding, err = NewCommandLineBinding(inputBinding, context)
		if err != nil {
			err = fmt.Errorf("(NewInputRecordFieldFromInterface) NewCWLTypeArray returned: %s", err.Error())
			return
		}
	}

	return
}

func CreateCommandInputRecordFieldArray(native []interface{}, schemata []CWLType_Type, context *WorkflowContext) (irfa []CommandInputRecordField, err error) {

	for _, elem := range native {

		var irf *CommandInputRecordField
		irf, err = NewCommandInputRecordField(elem, schemata, context)
		if err != nil {
			err = fmt.Errorf("(CreateInputRecordFieldArray) returned: %s", err.Error())
			return

		}

		irfa = append(irfa, *irf)
	}

	return
}
