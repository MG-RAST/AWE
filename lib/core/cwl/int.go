package cwl

import (
	"strconv"
)

type Int struct {
	Id    string `yaml:"id"`
	Value int    `yaml:"value"`
}

func (i Int) GetClass() string { return "Int" }
func (i Int) GetId() string    { return i.Id }
func (i Int) String() string   { return strconv.Itoa(i.Value) }
