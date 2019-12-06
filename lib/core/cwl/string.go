package cwl

import (
	"fmt"
	//"github.com/mitchellh/mapstructure"
)

//type String struct {
//	CWLType_Impl `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
//	//Class        CWLType_Type `yaml:"class,omitempty" json:"class,omitempty" bson:"class,omitempty"`
//	Value string `yaml:"value,omitempty" json:"value,omitempty" bson:"value,omitempty"`
//}

// String _
type String string

// IsCWLObject _
func (s *String) IsCWLObject() {}

// GetClass _
func (s *String) GetClass() string { return string(CWLString) } // for CWLObject

// GetType _
func (s *String) GetType() CWLType_Type { return CWLString }
func (s *String) String() string        { return string(*s) }

// GetID _
func (s *String) GetID() string { return "" }

// SetID _
func (s *String) SetID(i string) {}

// IsCWLMinimal _
func (s *String) IsCWLMinimal() {}

// NewString _
func NewString(value string) (s *String) {

	var sNptr String
	sNptr = String(value)

	s = &sNptr

	return

}

// NewStringFromInterface _
func NewStringFromInterface(native interface{}) (s *String, err error) {

	realString, ok := native.(string)
	if !ok {
		err = fmt.Errorf("(NewStringFromInterface) Cannot create string")
		return
	}
	s = NewString(realString)

	return
}
