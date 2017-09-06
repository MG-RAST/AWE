package cwl

type Boolean struct {
	CWLType_Impl `bson:",inline" json:",inline" mapstructure:",squash"`
	Value        bool `yaml:"value,omitempty" json:"value,omitempty" bson:"value,omitempty"`
}

func (s *Boolean) GetClass() string { return CWL_boolean } // for CWL_object
func (s *Boolean) String() string {
	if s.Value {
		return "True"
	}
	return "False"
}

func (s *Boolean) Is_CommandInputParameterType() {} // for CommandInputParameterType

func NewBoolean(id string, value bool) *Boolean {
	return &Boolean{CWLType_Impl: CWLType_Impl{Id: id}, Value: value}
}
