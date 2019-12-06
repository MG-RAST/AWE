package cwl

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
)

// Empty this is a generic CWLObject. Its only purpose is to retrieve the value of "class"
type Empty struct {
	CWLType_Impl `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
	ID           string `yaml:"id,omitempty" json:"id,omitempty" bson:"id,omitempty"`
	Class        string `yaml:"class,omitempty" json:"class,omitempty" bson:"class,omitempty"`
}

// GetClass _
func (e Empty) GetClass() string { return e.Class }

// GetID _
func (e Empty) GetID() string { return e.ID }

func (e Empty) String() string { return "Empty" }

// NewEmpty _
func NewEmpty(value interface{}) (obj_empty *Empty, err error) {
	obj_empty = &Empty{}

	//value_map, is_map := value.(map[string]interface{})
	//if is_map {
	//	value_map["type"] = CWLNull
	//}

	err = mapstructure.Decode(value, &obj_empty)
	if err != nil {
		fmt.Println("(NewEmpty) value: ")
		spew.Dump(value)
		err = fmt.Errorf("(NewEmpty) Could not convert into CWL object: %s", err.Error())
		return
	}

	return
}

// GetClass _
func GetClass(native interface{}) (class string, err error) {
	empty, xerr := NewEmpty(native)
	if xerr != nil {

		err = fmt.Errorf("(GetClass) NewEmpty returned: %s", xerr.Error())
		return
	}
	class = empty.GetClass()

	return
}

// GetID _
func GetID(native interface{}) (id string, err error) {
	empty, xerr := NewEmpty(native)
	if xerr != nil {
		err = fmt.Errorf("(GetId) NewEmpty returned: %s", xerr.Error())
		return
	}

	id = empty.GetID()

	if id == "" {
		//spew.Dump(native)
		err = fmt.Errorf("(empty/GetId) no id found")
		return
	}

	return
}
