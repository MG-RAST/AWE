package cwl

import (
	"fmt"
	"reflect"
	//"github.com/davecgh/go-spew/spew"
	//"reflect"
)

//https://www.commonwl.org/v1.0/Workflow.html#OutputRecordField
type OutputRecordField struct {
	RecordField   `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // name type, doc , label
	OutputBinding *CommandOutputBinding                                                 `yaml:"outputBinding,omitempty" json:"outputBinding,omitempty" bson:"outputBinding,omitempty"`
}

func NewOutputRecordField(native interface{}, schemata []CWLType_Type, context *WorkflowContext) (crf *OutputRecordField, err error) {

	native, err = MakeStringMap(native, context)
	if err != nil {
		return
	}

	var rf *RecordField
	rf, err = NewRecordFieldFromInterface(native, "", schemata, "Output", context)
	if err != nil {
		err = fmt.Errorf("(NewOutputRecordField) NewRecordFieldFromInterface returned: %s", err.Error())
		return
	}

	crf = &OutputRecordField{}
	crf.RecordField = *rf

	nativeMap, ok := native.(map[string]interface{})
	if !ok {
		err = fmt.Errorf("(NewOutputRecordField) type assertion error, got %s", reflect.TypeOf(native))
		return
	}

	outputBinding, has_outputBinding := nativeMap["outputBinding"]
	if has_outputBinding {

		crf.OutputBinding, err = NewCommandOutputBinding(outputBinding, context)
		if err != nil {
			err = fmt.Errorf("(NewOutputRecordField) NewCWLTypeArray returned: %s", err.Error())
			return
		}
	}

	return
}

func CreateOutputRecordFieldArray(native []interface{}, schemata []CWLType_Type, context *WorkflowContext) (irfa []OutputRecordField, err error) {

	for _, elem := range native {

		var irf *OutputRecordField
		irf, err = NewOutputRecordField(elem, schemata, context)
		if err != nil {
			err = fmt.Errorf("(CreateOutputRecordFieldArray) NewOutputRecordField returned: %s", err.Error())
			return

		}

		irfa = append(irfa, *irf)
	}

	return
}
