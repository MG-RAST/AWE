package cwl

import (
	"fmt"
)

type Any interface {
	CWL_object
	String() string
}

func NewAny(native interface{}) (any Any, err error) {

	switch native.(type) {
	case int:
		native_int := native.(int)

		any = &Int{Value: native_int}
	case string:
		native_str := native.(string)

		any = &String{Value: native_str}
	case bool:
		native_bool := native.(bool)

		any = &Boolean{Value: native_bool}

	default:
		err = fmt.Errorf("(NewAny) Type unknown")

	}
	return

}
