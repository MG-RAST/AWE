package cwl

// this is a generic CWL_object. Its only purpose is to retrieve the value of "class"
type Empty struct {
	Id    string `yaml:"id"`
	Class string `yaml:"class"`
}

func (e Empty) GetClass() string { return e.Class }
func (e Empty) GetId() string    { return e.Id }
func (e Empty) String() string   { return "Empty" }
