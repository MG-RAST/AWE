package cwl

// NamedSchemaIf _
type NamedSchemaIf interface {
	GetName() string
	SetName(s string)
}

// NamedSchema _
type NamedSchema struct {
	Schema `bson:",inline" yaml:",inline" json:",inline" mapstructure:",squash"` // provides Type and Label
	Name   string                                                                `yaml:"name,omitempty" json:"name,omitempty" bson:"name,omitempty"`
}

// GetName _
func (ns *NamedSchema) GetName() string { return ns.Name }

// SetName _
func (ns *NamedSchema) SetName(s string) {
	ns.Name = s
	return
}
