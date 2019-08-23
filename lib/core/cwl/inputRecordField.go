package cwl

import (
	"fmt"
	//"github.com/davecgh/go-spew/spew"
	"reflect"
)

// InputRecordField http://www.commonwl.org/v1.0/CommandLineTool.html#InputRecordField
type InputRecordField struct {
	RecordField  `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // provides: name, type, doc, label
	InputBinding *CommandLineBinding                                                   `yaml:"inputBinding,omitempty" json:"inputBinding,omitempty" bson:"inputBinding,omitempty"`
}

// NewInputRecordField _
func NewInputRecordField(native interface{}, name string, schemata []CWLType_Type, context *WorkflowContext) (irf *InputRecordField, err error) {

	native, err = MakeStringMap(native, context)
	if err != nil {
		return
	}

	//fmt.Println("(NewInputRecordField) native")
	//spew.Dump(native)

	var rf *RecordField
	rf, err = NewRecordFieldFromInterface(native, name, schemata, "Input", context)
	if err != nil {
		err = fmt.Errorf("(NewInputRecordField) NewRecordFieldFromInterface returned: %s", err.Error())
		return
	}

	irf = &InputRecordField{}
	irf.RecordField = *rf

	nativeMap, ok := native.(map[string]interface{})
	if ok {
		inputBinding_if, has_inputBinding := nativeMap["inputBinding"]
		if has_inputBinding {

			var inputBinding *CommandLineBinding
			inputBinding, err = NewCommandLineBinding(inputBinding_if, context)
			if err != nil {
				err = fmt.Errorf("(NewInputRecordField) NewCommandLineBinding returns: %s", err.Error())
				return
			}

			irf.InputBinding = inputBinding

		}
	}

	if name != "" {
		irf.Name = name
	}

	return
}

// CreateInputRecordFieldArray _
func CreateInputRecordFieldArray(native interface{}, schemata []CWLType_Type, context *WorkflowContext) (irfa []InputRecordField, err error) {

	native, err = MakeStringMap(native, context)
	if err != nil {
		return
	}

	//fmt.Println("CreateInputRecordFieldArray:")
	//spew.Dump(native)

	switch native.(type) {
	case []interface{}:
		nativeArray := native.([]interface{})
		for _, elem := range nativeArray {

			var irf *InputRecordField
			irf, err = NewInputRecordField(elem, "", schemata, context)
			if err != nil {
				err = fmt.Errorf("(CreateInputRecordFieldArray) NewInputRecordField returned: %s", err.Error())
				return

			}

			irfa = append(irfa, *irf)
		}

	case map[string]interface{}:
		nativeMap := native.(map[string]interface{})

		for name, elem := range nativeMap {

			//fmt.Printf("\n\nelem: %s\n\n", name)
			//spew.Dump(elem)
			//fmt.Println("\n-------\n\n")
			var irf *InputRecordField
			irf, err = NewInputRecordField(elem, name, schemata, context)
			if err != nil {
				err = fmt.Errorf("(CreateInputRecordFieldArray) NewInputRecordField returned: %s", err.Error())
				return

			}

			irfa = append(irfa, *irf)
		}

	default:
		err = fmt.Errorf("(CreateInputRecordFieldArray) type not supported: %s", reflect.TypeOf(native))
	}
	return
}
