package cwl

import (
	"fmt"
	//"github.com/davecgh/go-spew/spew"
	"reflect"
)

// http://www.commonwl.org/v1.0/CommandLineTool.html#InputRecordField
type InputRecordField struct {
	RecordField  `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // provides: name, type, doc, label
	InputBinding *CommandLineBinding                                                   `yaml:"inputBinding,omitempty" json:"inputBinding,omitempty" bson:"inputBinding,omitempty"`
}

func NewInputRecordField(native interface{}, schemata []CWLType_Type, context *WorkflowContext) (irf *InputRecordField, err error) {

	native, err = MakeStringMap(native, context)
	if err != nil {
		return
	}

	var rf *RecordField
	rf, err = NewRecordFieldFromInterface(native, schemata, "Input", context)
	if err != nil {
		err = fmt.Errorf("(NewInputRecordField) NewRecordFieldFromInterface returned: %s", err.Error())
		return
	}

	irf = &InputRecordField{}
	irf.RecordField = *rf

	native_map, ok := native.(map[string]interface{})
	if !ok {
		err = fmt.Errorf("(NewInputRecordField) type not supported: %s", reflect.TypeOf(native))
		return
	}

	inputBinding_if, has_inputBinding := native_map["inputBinding"]
	if has_inputBinding {

		var inputBinding *CommandLineBinding
		inputBinding, err = NewCommandLineBinding(inputBinding_if, context)
		if err != nil {
			err = fmt.Errorf("(NewInputRecordField) NewCommandLineBinding returns: %s", err.Error())
			return
		}

		irf.InputBinding = inputBinding

	}

	return
}

func CreateInputRecordFieldArray(native interface{}, schemata []CWLType_Type, context *WorkflowContext) (irfa []InputRecordField, err error) {

	switch native.(type) {
	case []interface{}:
		native_array := native.([]interface{})
		for _, elem := range native_array {

			var irf *InputRecordField
			irf, err = NewInputRecordField(elem, schemata, context)
			if err != nil {
				err = fmt.Errorf("(CreateInputRecordFieldArray) returned: %s", err.Error())
				return

			}

			irfa = append(irfa, *irf)
		}
	default:
		err = fmt.Errorf("(CreateInputRecordFieldArray) type not supported: %s", reflect.TypeOf(native))
	}
	return
}
