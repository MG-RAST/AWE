package core

//type App struct {
//	Name     string        `bson:"name" json:"name"`
//	App_args []AppResource `bson:"app_args" json:"app_args"`

//	AppDef *AppCommandMode `bson:"appdef" json:"appdef"` // App defintion

//}

// Command _
type Command struct {
	Name          string   `bson:"name" json:"name" mapstructure:"name"`
	Args          string   `bson:"args" json:"args" mapstructure:"args"`
	ArgsArray     []string `bson:"args_array" json:"args_array" mapstructure:"args_array"`    // use this instead of Args, which is just a string
	Dockerimage   string   `bson:"Dockerimage" json:"Dockerimage" mapstructure:"Dockerimage"` // for Shock (TODO rename this !)
	DockerPull    string   `bson:"dockerPull" json:"dockerPull" mapstructure:"dockerPull"`    // docker pull
	XCmdScript    []string `bson:"cmd_script" json:"cmd_script" mapstructure:"cmd_script"`
	Environ       Envs     `bson:"environ" json:"environ" mapstructure:"environ"`
	HasPrivateEnv bool     `bson:"has_private_env" json:"has_private_env" mapstructure:"has_private_env"`
	Description   string   `bson:"description" json:"description" mapstructure:"description"`
	ParsedArgs    []string `bson:"-" json:"-" mapstructure:"-"`
	Local         bool     // indicates local execution, i.e. working directory is same as current working directory (do not delete !)
}

// Envs _
type Envs struct {
	Public  map[string]string `bson:"public" json:"public"`
	Private map[string]string `bson:"private" json:"-"`
}

// NewCommand _
func NewCommand(name string) *Command {
	return &Command{
		Name: name,
	}
}

// EnvironP following special code is in order to unmarshal the private field Command.Environ.Private,
//  so put them in to this file for less confusion
type EnvironP struct {
	Private map[string]string `json:"private"`
}

// CommandP _
type CommandP struct {
	Environ *EnvironP `json:"environ"`
}

// TaskP _
type TaskP struct {
	Cmd *CommandP `json:"cmd"`
}

// JobP _
type JobP struct {
	Tasks []*TaskP `json:"tasks"`
}
