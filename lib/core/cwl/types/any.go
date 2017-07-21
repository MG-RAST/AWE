package cwl

import (
	"fmt"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	"reflect"
)

//type Any interface {
//	CWL_object
//	String() string
//}

type Any interface {
	CWL_object
}

func NewAny(native interface{}) (any Any, err error) {

	cwl_type, err := NewCWLType(native)
	if err != nil {
		fmt.Println("cwl_type: ")
		fmt.Println(reflect.TypeOf(cwl_type))
		any = nil
		return
	}

	any = cwl_type

	//TODO File, Directory

	return

}
