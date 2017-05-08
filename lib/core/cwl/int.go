package cwl

import (
	"fmt"
	"strconv"
)

type Int struct {
	CWLType_Impl
	Id    string `yaml:"id"`
	Value int    `yaml:"value"`
}

func (i *Int) GetClass() string { return "Int" }
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
