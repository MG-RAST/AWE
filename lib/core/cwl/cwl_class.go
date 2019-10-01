package cwl

//"fmt"
//"reflect"

type CWL_class interface {
	GetClass() string
}

type CWL_class_Impl struct {
	Class string `yaml:"class,omitempty" json:"class,omitempty" bson:"class,omitempty"`
}

func (c *CWL_class_Impl) GetClass() string { return c.Class }
