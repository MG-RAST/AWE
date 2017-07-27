package cwl

import ()

type Enum struct {
	CWLType_Impl
	Id      string
	Symbols []string
}

func (e *Enum) GetClass() string { return CWL_enum }
func (e *Enum) GetId() string    { return e.Id }
func (e *Enum) SetId(id string)  { e.Id = id }
