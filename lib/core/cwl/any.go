package cwl

import (
	"fmt"

	"reflect"
)

//
// *** not used ***
//

//type Any interface {
//	CWLObject
//	String() string
//}

// Any _
type Any interface {
	CWL_class
}

// NewAny _
func NewAny(native interface{}, context *WorkflowContext) (any Any, err error) {

	cwlType, err := NewCWLType("", native, context)
	if err != nil {
		fmt.Println("cwl_type: ")
		fmt.Println(reflect.TypeOf(cwlType))
		any = nil
		return
	}

	any = cwlType

	//TODO File, Directory

	return

}
