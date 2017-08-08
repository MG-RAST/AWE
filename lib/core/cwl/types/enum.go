package cwl

import ()

type Enum struct {
	CWLType_Impl
	Id      string   `yaml:"id,omitempty" json:"id,omitempty" bson:"id,omitempty"`
	Symbols []string `yaml:"symbols,omitempty" json:"symbols,omitempty" bson:"symbols,omitempty"`
}

func (e *Enum) GetClass() string { return CWL_enum }
func (e *Enum) GetId() string    { return e.Id }
func (e *Enum) SetId(id string)  { e.Id = id }
