package cwl

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	//"github.com/mitchellh/mapstructure"
)

type CommandOutputParameterType interface {
	CWL_minimal_interface
	is_CommandOutputParameterType()
}

type CommandOutputParameterType_Impl struct{}

func (c *CommandOutputParameterType_Impl) is_CommandOutputParameterType() {}

func NewCommandOutputParameterType(unparsed interface{}) (cipt_ptr *CommandOutputParameterType, err error) {

	// Try CWL_Type
	var cipt CommandOutputParameterType

	cwl_type, err := NewCWLType(unparsed) // returns *CWLType_Impl

	if err == nil {

		//cwl_type_x := *cwl_type

		cipt = cwl_type.(CommandOutputParameterType)
		cipt_ptr = &cipt
		return
	}

	spew.Dump(unparsed)
	err = fmt.Errorf("(NewCommandOutputParameterType) Type unknown")

	return

}
