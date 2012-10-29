package core

import (
	"github.com/MG-RAST/AWE/conf"
)

type Info struct {
	Name     string `bson:"name" json:"name"`
	Owner    string `bson:"owner" json:"owner"`
	State    string `bson:"state" json:"state"`
	Priority int    `bson:"priority" json:"priority"`
}

func NewInfo() *Info {
	return &Info{State: "Submitted", Priority: conf.BasePriority}
}
