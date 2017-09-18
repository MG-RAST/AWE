package cwl

type BaseRequirement struct {
	Class string `yaml:"class,omitempty" json:"class,omitempty" bson:"class,omitempty"`
}

func (c BaseRequirement) GetClass() string { return c.Class }
