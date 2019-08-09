package cwl

//"fmt"
//"reflect"

// IdentifierInterface _
type IdentifierInterface interface {
	//CWL_bbject_interface
	//GetClass() string
	GetID() string
	SetID(string)
	//is_Any()
}

// IdentifierImpl _
type IdentifierImpl struct {
	ID string `yaml:"id,omitempty" json:"id,omitempty" bson:"id,omitempty"`
}

// GetID _
func (c *IdentifierImpl) GetID() string { return c.ID }

// SetID _
func (c *IdentifierImpl) SetID(id string) { c.ID = id; return }
