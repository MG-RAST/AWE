package core

import (
	"regexp"
)

var (
	//<inputs::i1> <inputs::i2> <outputs::o1>
	TemplateRe = regexp.MustCompile("\\w+::\\w+")
)

type Command struct {
	Name        string   `bson:"name" json:"name"`
	Options     string   `bson:"options" json:"-"`
	Args        string   `bson:"args" json:"args"`
	Template    string   `bson:"template" json:"-"`
	Description string   `bson:"description" json:"description"`
	ParsedArgs  []string `bson:"-" json:"-"`
}

func (c *Command) Substitute(inputs *IOmap, outputs *IOmap) (err error) {
	if len(c.Template) == 0 {
		return
	}
	for _, s := range TemplateRe.FindAllString(c.Template, -1) {
		print(s + "\n")
	}
	return
}

func NewCommand(name string) *Command {
	return &Command{
		Name: name,
	}
}
