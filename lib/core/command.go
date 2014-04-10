package core

type Command struct {
	Name          string   `bson:"name" json:"name"`
	Args          string   `bson:"args" json:"args"`
	Dockerimage   string   `bson:"Dockerimage" json:"Dockerimage"`
	Environ       Envs     `bson:"environ" json:"environ"`
	HasPrivateEnv bool     `bson:"has_private_env" json:"has_private_env"`
	Description   string   `bson:"description" json:"description"`
	ParsedArgs    []string `bson:"-" json:"-"`
}

type Envs struct {
	Public  map[string]string `bson:"public" json:"public"`
	Private map[string]string `bson:"private" json:"-"`
}

func NewCommand(name string) *Command {
	return &Command{
		Name: name,
	}
}

//following special code is in order to unmarshal the private field Command.Environ.Private,
//so put them in to this file for less confusion
type Environ_p struct {
	Private map[string]string `json:"private"`
}

type Command_p struct {
	Environ *Environ_p `json:"environ"`
}

type Task_p struct {
	Cmd *Command_p `json:"cmd"`
}

type Job_p struct {
	Tasks []*Task_p `json:"tasks"`
}
