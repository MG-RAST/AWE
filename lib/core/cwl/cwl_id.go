package cwl

//"fmt"
//"reflect"

// CWL_id _
type CWL_id interface {
	//CWL_bbject_interface
	//GetClass() string
	GetID() string
	SetID(string)
	//is_Any()
}

// CWL_id_Impl _
type CWL_id_Impl struct {
	ID string `yaml:"id,omitempty" json:"id,omitempty" bson:"id,omitempty"`
}

// GetID _
func (c *CWL_id_Impl) GetID() string { return c.ID }

// SetID _
func (c *CWL_id_Impl) SetID(id string) { c.ID = id; return }
