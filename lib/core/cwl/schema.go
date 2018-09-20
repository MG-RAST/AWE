package cwl

type Schema struct {
	Type  CWLType_Type `yaml:"type,omitempty" json:"type,omitempty" bson:"type,omitempty"`
	Label string       `yaml:"label,omitempty" json:"label,omitempty" bson:"label,omitempty"`
}
