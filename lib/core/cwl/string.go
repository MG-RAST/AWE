package cwl

type String struct {
	Id    string `yaml:"id"`
	Value string `yaml:"value"`
}

func (s String) GetClass() string { return "String" } // for CWL_object
func (s String) GetId() string    { return s.Id }     // for CWL_object
func (s String) String() string   { return s.Value }
func (s String) is_CWLType()      {} // for CWLType
