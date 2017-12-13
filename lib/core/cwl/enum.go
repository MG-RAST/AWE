package cwl

import ()

type Enum string

//type Enum struct {
//	CWLType_Impl `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
//	Symbols      []string `yaml:"symbols,omitempty" json:"symbols,omitempty" bson:"symbols,omitempty"`
//}

func (e *Enum) GetClass() string { return "enum" }
