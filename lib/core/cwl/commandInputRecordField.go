package cwl

import (
	"fmt"
	//"github.com/davecgh/go-spew/spew"
	//"reflect"
)

// http://www.commonwl.org/v1.0/CommandLineTool.html#InputRecordField
type CommandInputRecordField struct {
	RecordField `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
}

func NewCommandInputRecordField(native interface{}, schemata []CWLType_Type) (crf *CommandInputRecordField, err error) {
	var rf *RecordField
	rf, err = NewRecordFieldFromInterface(native, schemata)
	if err != nil {
		return
	}

	crf = &CommandInputRecordField{}
	crf.RecordField = *rf

	return
}

func CreateCommandInputRecordFieldArray(native []interface{}, schemata []CWLType_Type) (irfa []CommandInputRecordField, err error) {

	for _, elem := range native {

		var irf *CommandInputRecordField
		irf, err = NewCommandInputRecordFieldFromInterface(elem, schemata)
		if err != nil {
			err = fmt.Errorf("(CreateInputRecordFieldArray) returned: %s", err.Error())
			return

		}

		irfa = append(irfa, *irf)
	}

	return
}
