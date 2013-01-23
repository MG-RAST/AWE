package core

import (
	"github.com/MG-RAST/AWE/conf"
	"time"
)

type Info struct {
	Name       string    `bson:"name" json:"name"`
	Owner      string    `bson:"owner" json:"owner"`
	Clients    string    `bson:"clients" json:"clients"`
	SubmitTime time.Time `bson:"submittime" json:"submittime"`
	Priority   int       `bson:"priority" json:"priority"`
}

func NewInfo() *Info {
	return &Info{
		SubmitTime: time.Now(),
		Priority:   conf.BasePriority,
	}
}
