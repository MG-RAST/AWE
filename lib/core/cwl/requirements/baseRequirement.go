package requirements

type BaseRequirement struct {
	Class string `yaml:"class" bson:"class" json:"class"`
}

func (c BaseRequirement) GetClass() string { return c.Class }
