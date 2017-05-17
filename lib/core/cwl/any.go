package cwl

import (
	"fmt"
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
	if err == nil {
		fmt.Println("cwl_type: ")
		fmt.Println(reflect.TypeOf(cwl_type))
		any = nil
		return
	}

	err = fmt.Errorf("(NewAny) type unknown")

	//TODO File, Directory

	return

}
