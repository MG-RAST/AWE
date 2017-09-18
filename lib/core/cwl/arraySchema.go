package cwl

import (
//"fmt"
)

type ArraySchema struct {
	Items []CWLType_Type `yaml:"items,omitempty" bson:"items,omitempty" json:"items,omitempty"` // string or []string ([] speficies which types are possible, e.g ["File" , "null"])
	Type  string         `yaml:"type,omitempty" bson:"type,omitempty" json:"type,omitempty"`    // must be array
	Label string         `yaml:"label,omitempty" bson:"label,omitempty" json:"label,omitempty"`
}
