package cwl

import (
//"fmt"
//"reflect"
)

type CWL_object interface {
	Is_CWL_object()
}

type CWL_object_Impl struct{}

func (c *CWL_object_Impl) Is_CWL_object() {}
