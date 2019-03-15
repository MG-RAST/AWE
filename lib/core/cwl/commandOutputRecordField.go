package cwl

import (
	"fmt"
	"reflect"
	//"github.com/davecgh/go-spew/spew"
	//"reflect"
)

//https://www.commonwl.org/v1.0/CommandLineTool.html#CommandOutputRecordField
type CommandOutputRecordField struct {
	RecordField   `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // name type, doc , label
	OutputBinding *CommandOutputBinding                                                 `yaml:"outputBinding,omitempty" json:"outputBinding,omitempty" bson:"outputBinding,omitempty"`
	//RecordField `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
}

func NewCommandOutputRecordField(native interface{}, schemata []CWLType_Type, context *WorkflowContext) (crf *CommandOutputRecordField, err error) {

	native, err = MakeStringMap(native, context)
	if err != nil {
		return
	}

	var rf *RecordField
	rf, err = NewRecordFieldFromInterface(native, schemata, "CommandOutput", context)
	if err != nil {
		err = fmt.Errorf("(NewCommandOutputRecordField) NewRecordFieldFromInterface returned: %s", err.Error())
		return
	}

	crf = &CommandOutputRecordField{}
	crf.RecordField = *rf

	nativeMap, ok := native.(map[string]interface{})
	if !ok {
		err = fmt.Errorf("(NewCommandOutputRecordField) type assertion error, got %s", reflect.TypeOf(native))
		return
	}

	outputBinding, has_outputBinding := nativeMap["outputBinding"]
	if has_outputBinding {

		crf.OutputBinding, err = NewCommandOutputBinding(outputBinding, context)
		if err != nil {
			err = fmt.Errorf("(NewCommandOutputRecordField) NewCWLTypeArray returned: %s", err.Error())
			return
		}
	}

	return
}

func CreateCommandOutputRecordFieldArray(native []interface{}, schemata []CWLType_Type, context *WorkflowContext) (irfa []CommandOutputRecordField, err error) {

	for _, elem := range native {

		var irf *CommandOutputRecordField
		irf, err = NewCommandOutputRecordField(elem, schemata, context)
		if err != nil {
			err = fmt.Errorf("(CreateCommandOutputRecordFieldArray) returned: %s", err.Error())
			return

		}

		irfa = append(irfa, *irf)
	}

	return
}
