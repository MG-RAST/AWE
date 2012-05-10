package core

import ()

type Workunit struct {
	Info    Info    `bson:"info" json:"info"`
	Inputs  IOmap   `bson:"inputs" json:"inputs"`
	Outputs IOmap   `bson:"outputs" json:"outputs"`
	Cmd     Command `bson:"cmd" json:"cmd"`
}
