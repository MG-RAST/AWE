package cwl

import (
	"fmt"
	//"github.com/davecgh/go-spew/spew"
	//"reflect"
)

//https://www.commonwl.org/v1.0/CommandLineTool.html#CommandOutputRecordField
type CommandOutputRecordField struct {
	RecordField   `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // name type, doc , label
	OutputBinding *CommandOutputBinding
	//RecordField `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
}

func NewCommandOutputRecordField(native interface{}, schemata []CWLType_Type) (crf *CommandOutputRecordField, err error) {
	var rf *RecordField
	rf, err = NewRecordFieldFromInterface(native, schemata)
	if err != nil {
		return
	}

	crf = &CommandOutputRecordField{}
	crf.RecordField = *rf

	native_map, ok := native.(map[string]interface{})
	if !ok {
		err = fmt.Errorf("(NewCommandOutputRecordField) type assertion error")
		return
	}

	outputBinding, has_outputBinding := native_map["outputBinding"]
	if has_outputBinding {

		crf.OutputBinding, err = NewCommandOutputBinding(outputBinding)
		if err != nil {
			err = fmt.Errorf("(NewCommandOutputRecordField) NewCWLTypeArray returned: %s", err.Error())
			return
		}
	}

	return
}

func CreateCommandOutputRecordFieldArray(native []interface{}, schemata []CWLType_Type) (irfa []CommandOutputRecordField, err error) {

	for _, elem := range native {

		var irf *CommandOutputRecordField
		irf, err = NewCommandOutputRecordField(elem, schemata)
		if err != nil {
			err = fmt.Errorf("(CreateCommandOutputRecordFieldArray) returned: %s", err.Error())
			return

		}

		irfa = append(irfa, *irf)
	}

	return
}
