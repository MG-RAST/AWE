package cwl

type Boolean struct {
	CWLType_Impl
	Id    string `yaml:"id,omitempty" json:"id,omitempty" bson:"id,omitempty"`
	Value bool   `yaml:"value,omitempty" json:"value,omitempty" bson:"value,omitempty"`
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

func (s *Boolean) Is_CommandInputParameterType() {} // for CommandInputParameterType
