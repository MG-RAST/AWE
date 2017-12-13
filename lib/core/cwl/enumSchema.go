package cwl

import ()

type EnumSchema struct {
	Symbols []string `yaml:"symbols,omitempty" bson:"symbols,omitempty" json:"symbols,omitempty"`
	Type    string   `yaml:"type,omitempty" bson:"type,omitempty" json:"type,omitempty"` // must be enum
	Label   string   `yaml:"label,omitempty" bson:"label,omitempty" json:"label,omitempty"`
}
