package cwl

type NamedSchema struct {
	Schema `bson:",inline" yaml:",inline" json:",inline" mapstructure:",squash"` // provides Type and Label
	Name   string                                                                `yaml:"name,omitempty" json:"name,omitempty" bson:"name,omitempty"`
}
