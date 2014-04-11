package core

import (
	"github.com/MG-RAST/AWE/lib/conf"
	"time"
)

//job info
type Info struct {
	Name          string    `bson:"name" json:"name"`
	Xref          string    `bson:"xref" json:"xref"`
	Project       string    `bson:"project" json:"project"`
	User          string    `bson:"user" json:"user"`
	Pipeline      string    `bson:"pipeline" json:"pipeline"`
	ClientGroups  string    `bson:"clientgroups" json:"clientgroups"`
	SubmitTime    time.Time `bson:"submittime" json:"submittime"`
	StartedTme    time.Time `bson:"startedtime" json:"startedtime"`
	CompletedTime time.Time `bson:"completedtime" json:"completedtime"`
	Priority      int       `bson:"priority" json:"priority"`
	Auth          bool      `bson:"auth" json:"auth"`
	DataToken     string    `bson:"datatoken" json:"-"`
	NoRetry       bool      `bson:"noretry" json:"noretry"`
}

func NewInfo() *Info {
	return &Info{
		SubmitTime: time.Now(),
		Priority:   conf.BasePriority,
	}
}
