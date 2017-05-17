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
	is_CommandOutputParameterType()
	is_CWLType()
	//is_CWL_minimal()
}

type CWLType_Impl struct{}

func (c *CWLType_Impl) is_CWL_minimal()                {}
func (c *CWLType_Impl) is_CWLType()                    {}
func (c *CWLType_Impl) is_CommandInputParameterType()  {}
func (c *CWLType_Impl) is_CommandOutputParameterType() {}

func NewCWLType(native interface{}) (cwl_type CWLType, err error) {

	//var cwl_type CWLType

	switch native.(type) {
	case int:
		native_int := native.(int)

		cwl_type = &Int{Value: native_int}

		//cwl_type = int_type.(*CWLType)

		return
		//temp = &Int{Value: native_int}

	case string:
		native_str := native.(string)

		cwl_type = &String{Value: native_str}
	case bool:
		native_bool := native.(bool)

		cwl_type = &Boolean{Value: native_bool}

	case map[interface{}]interface{}:
		empty, xerr := NewEmpty(native)
		if xerr != nil {
			err = xerr
			return
		}
		switch empty.GetClass() {
		case "File":
			file, yerr := NewFile(native)
			cwl_type = &file
			if yerr != nil {
				err = yerr
				return
			}
		default:
			err = fmt.Errorf("(NewCWLType) Map type unknown")
			return
		}

	default:

		err = fmt.Errorf("(NewCWLType) Type unknown")
		return
	}
	//cwl_type_ptr = &cwl_type

	return

}
