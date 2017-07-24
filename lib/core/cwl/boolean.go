package cwl

type Boolean struct {
	CWLType_Impl
	Id    string `yaml:"id"`
	Value bool   `yaml:"value"`
}

func (s *Boolean) GetClass() string { return CWL_boolean } // for CWL_object
func (s *Boolean) GetId() string    { return s.Id }        // for CWL_object
func (s *Boolean) SetId(id string)  { s.Id = id }
func (s *Boolean) String() string {
	if s.Value {
		return "True"
	}
	return "False"
}

func (s *Boolean) is_CommandInputParameterType() {} // for CommandInputParameterType
