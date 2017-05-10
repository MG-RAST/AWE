package cwl

import (
	"fmt"
)

// CWLType - CWL basic types: int, string, boolean, .. etc
// http://www.commonwl.org/v1.0/CommandLineTool.html#CWLType
// null, boolean, int, long, float, double, string, File, Directory
type CWLType interface {
	CWL_object
	is_CommandInputParameterType()
	is_CWLType()
	//is_CWL_minimal()
}

type CWLType_Impl struct{}

func (c *CWLType_Impl) is_CWL_minimal()               {}
func (c *CWLType_Impl) is_CWLType()                   {}
func (c *CWLType_Impl) is_CommandInputParameterType() {}

func NewCWLType(native interface{}) (cwl_type *CWLType, err error) {

	var temp interface{}

	switch native.(type) {
	case int:
		native_int := native.(int)

		temp = Int{Value: native_int}

	case string:
		native_str := native.(string)

		temp = String{Value: native_str}
	case bool:
		native_bool := native.(bool)

		temp = Boolean{Value: native_bool}

	default:
		err = fmt.Errorf("(NewAny) Type unknown")

	}

	cwl_type = temp.(*CWLType)

	return

}
