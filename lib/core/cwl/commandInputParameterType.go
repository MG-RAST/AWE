package cwl

import (
	"fmt"
	//"github.com/mitchellh/mapstructure"
)

type CommandInputParameterType interface {
	CWL_minimal_interface
	is_CommandInputParameterType()
}

type CommandInputParameterType_Impl struct{}

func (c *CommandInputParameterType_Impl) is_CommandInputParameterType() {}

func NewCommandInputParameterType(unparsed interface{}) (cipt_ptr *CommandInputParameterType, err error) {

	// Try CWL_Type
	var cipt CommandInputParameterType

	cwl_type, err := NewCWLType(unparsed) // returns *CWLType_Impl

	if err == nil {

		//cwl_type_x := *cwl_type

		cipt = cwl_type.(CommandInputParameterType)
		cipt_ptr = &cipt
		return
	}

	err = fmt.Errorf("(NewCommandInputParameterType) Type unknown")

	return

}

func CreateCommandInputParameterTypeArray(v interface{}) (cipt_array_ptr *[]CommandInputParameterType, err error) {

	cipt_array := []CommandInputParameterType{}

	array, ok := v.([]interface{})

	if ok {
		//handle array case
		for _, v := range array {

			cipt, xerr := NewCommandInputParameterType(v)
			if xerr != nil {
				err = xerr
				return
			}

			cipt_array = append(cipt_array, *cipt)
		}
		cipt_array_ptr = &cipt_array
		return
	}

	// handle non-arrary case

	cipt, err := NewCommandInputParameterType(v)
	if err != nil {

		return
	}

	cipt_array = append(cipt_array, *cipt)
	cipt_array_ptr = &cipt_array

	return
}
