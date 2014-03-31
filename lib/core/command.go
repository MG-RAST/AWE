package core

type Command struct {
	Name        string            `bson:"name" json:"name"`
	Args        string            `bson:"args" json:"args"`
	Dockerimage string            `bson:"Dockerimage" json:"Dockerimage"`
	Environ     map[string]string `bson:"environ" json:"environ"`
	Description string            `bson:"description" json:"description"`
	ParsedArgs  []string          `bson:"-" json:"-"`
}

func NewCommand(name string) *Command {
	return &Command{
		Name: name,
	}
}
