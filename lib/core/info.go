package core

import (
	"github.com/MG-RAST/AWE/lib/conf"
	"time"
)

type Info struct {
	Name         string    `bson:"name" json:"name"`
	Xref         string    `bson:"xref" json:"xref"`
	Project      string    `bson:"project" json:"project"`
	User         string    `bson:"user" json:"user"`
	Pipeline     string    `bson:"pipeline" json:"pipeline"`
	ClientGroups string    `bson:"clientgroups" json:"clientgroups"`
	SubmitTime   time.Time `bson:"submittime" json:"submittime"`
	Priority     int       `bson:"priority" json:"-"`
	Auth         bool      `bson:"auth" json:"auth"`
	DataToken    string    `bson:"datatoken" json:"-"`
}

func NewInfo() *Info {
	return &Info{
		SubmitTime: time.Now(),
		Priority:   conf.BasePriority,
	}
}
