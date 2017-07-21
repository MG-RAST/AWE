package cwl

type String struct {
	CWLType_Impl
	Id    string `yaml:"id"`
	Value string `yaml:"value"`
}

func (s *String) GetClass() string { return CWL_string } // for CWL_object
func (s *String) GetId() string    { return s.Id }       // for CWL_object
func (s *String) SetId(id string)  { s.Id = id }
func (s *String) String() string   { return s.Value }
