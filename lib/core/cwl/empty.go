package cwl

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
)

// this is a generic CWL_object. Its only purpose is to retrieve the value of "class"
type Empty struct {
	Id    string `yaml:"id"`
	Class string `yaml:"class"`
}

func (e Empty) GetClass() string { return e.Class }
func (e Empty) GetId() string    { return e.Id }
func (e Empty) SetId(id string)  { e.Id = id }
func (e Empty) String() string   { return "Empty" }

func NewEmpty(value interface{}) (obj_empty *Empty, err error) {
	obj_empty = &Empty{}

	err = mapstructure.Decode(value, &obj_empty)
	if err != nil {
		err = fmt.Errorf("Could not convert into CWL object")
		return
	}

	return
}
