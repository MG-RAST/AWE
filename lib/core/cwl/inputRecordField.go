package cwl

import (
	"fmt"
	//"github.com/davecgh/go-spew/spew"
	//"reflect"
)

// http://www.commonwl.org/v1.0/CommandLineTool.html#InputRecordField
type InputRecordField struct {
	RecordField `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
}

func NewInputRecordField(native interface{}, schemata []CWLType_Type) (irf *InputRecordField, err error) {
	var rf *RecordField
	rf, err = NewRecordFieldFromInterface(native, schemata)
	if err != nil {
		err = fmt.Errorf("(NewInputRecordField) NewRecordFieldFromInterface returned: %s", err.Error())
		return
	}

	irf = &InputRecordField{}
	irf.RecordField = *rf

	return
}

func CreateInputRecordFieldArray(native []interface{}, schemata []CWLType_Type) (irfa []InputRecordField, err error) {

	for _, elem := range native {

		var irf *InputRecordField
		irf, err = NewInputRecordField(elem, schemata)
		if err != nil {
			err = fmt.Errorf("(CreateInputRecordFieldArray) returned: %s", err.Error())
			return

		}

		irfa = append(irfa, *irf)
	}

	return
}
