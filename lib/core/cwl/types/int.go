package cwl

import (
	"fmt"
	"strconv"
)

type Int struct {
	CWLType_Impl
	Id    string `yaml:"id,omitempty" json:"id,omitempty" bson:"id,omitempty"`
	Value int    `yaml:"value,omitempty" json:"value,omitempty" bson:"value,omitempty"`
}

func (i *Int) GetClass() string { return CWL_int }
func (i *Int) GetId() string {
	fmt.Printf("GetId id=%s\n", i.Id)
	return i.Id
}
func (i *Int) SetId(id string) {
	fmt.Printf("SetId id=%s\n", id)
	fmt.Println("Hello world")
	i.Id = id
	fmt.Printf("SetId i.Id=%s\n", i.Id)
}
func (i *Int) String() string { return strconv.Itoa(i.Value) }
