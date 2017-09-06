package cwl

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
)

// this is a generic CWL_object. Its only purpose is to retrieve the value of "class"
type Empty struct {
	CWLType_Impl `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
	Class        string `yaml:"class,omitempty" json:"class,omitempty" bson:"class,omitempty"`
}

func (e Empty) GetClass() string { return e.Class }

func (e Empty) String() string { return "Empty" }

func NewEmpty(value interface{}) (obj_empty *Empty, err error) {
	obj_empty = &Empty{}

	err = mapstructure.Decode(value, &obj_empty)
	if err != nil {
		err = fmt.Errorf("(NewEmpty) Could not convert into CWL object: %s", err.Error())
		return
	}

	return
}

func GetClass(native interface{}) (class string, err error) {
	empty, xerr := NewEmpty(native)
	if xerr != nil {
		err = xerr
		return
	}
	class = empty.GetClass()

	return
}

func GetId(native interface{}) (id string, err error) {
	empty, xerr := NewEmpty(native)
	if xerr != nil {
		err = xerr
		return
	}
	id = empty.GetId()

	if id == "" {
		spew.Dump(native)
		panic("class name empty")
	}

	return
}
