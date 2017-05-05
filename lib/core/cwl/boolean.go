package cwl

type Boolean struct {
	Id    string `yaml:"id"`
	Value bool   `yaml:"value"`
}

func (s *Boolean) GetClass() string { return "Boolean" } // for CWL_object
func (s *Boolean) GetId() string    { return s.Id }      // for CWL_object
func (s *Boolean) SetId(id string)  { s.Id = id }
func (s *Boolean) String() string {
	if s.Value {
		return "True"
	}
	return "False"
}
func (s *Boolean) is_CWLType() {} // for CWLType
