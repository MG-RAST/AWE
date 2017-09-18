package cwl

type Boolean struct {
	CWLType_Impl `bson:",inline" json:",inline" mapstructure:",squash"`
	Value        bool `yaml:"value,omitempty" json:"value,omitempty" bson:"value,omitempty"`
}

func (s *Boolean) GetClass() string      { return string(CWL_boolean) } // for CWL_object
func (s *Boolean) GetType() CWLType_Type { return CWL_boolean }
func (s *Boolean) String() string {
	if s.Value {
		return "True"
	}
	return "False"
}

func (s *Boolean) Is_CommandInputParameterType() {} // for CommandInputParameterType

func NewBoolean(id string, value bool) *Boolean {
	b := &Boolean{}
	b.Class = "boolean"
	b.Type = CWL_boolean
	b.Id = id
	b.Value = value
	return b
}
