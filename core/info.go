package core

import (
	"github.com/MG-RAST/AWE/conf"
	"time"
)

type Info struct {
	Name         string    `bson:"name" json:"name"`
	Project      string    `bson:"project" json:"project"`
	User         string    `bson:"user" json:"user"`
	Pipeline     string    `bson:"pipeline" json:"pipeline"`
	ClientGroups string    `bson:"clientgroups" json:"clientgroups"`
	SubmitTime   time.Time `bson:"submittime" json:"submittime"`
	Priority     int       `bson:"priority" json:"-"`
}

func NewInfo() *Info {
	return &Info{
		SubmitTime: time.Now(),
		Priority:   conf.BasePriority,
	}
}
