package core

type Command struct {
<<<<<<< HEAD
	Dockerimage string   `bson:"Dockerimage" json:"Dockerimage"`
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
=======
	Name        string            `bson:"name" json:"name"`
	Args        string            `bson:"args" json:"args"`
	Dockerimage string   		   `bson:"Dockerimage" json:"Dockerimage"`
	Environ     map[string]string `bson:"environ" json:"environ"`
	Description string            `bson:"description" json:"description"`
	ParsedArgs  []string          `bson:"-" json:"-"`
>>>>>>> ce578bed8944f29dade6b9700b5cea01457b8129
}

func NewCommand(name string) *Command {
	return &Command{
		Name: name,
	}
}
