package core

import (
	"github.com/MG-RAST/AWE/lib/conf"
	"time"
)

//job info
type Info struct {
	Name          string            `bson:"name" json:"name" mapstructure:"name"`
	Xref          string            `bson:"xref" json:"xref" mapstructure:"xref"`
	Service       string            `bson:"service" json:"service" mapstructure:"service"`
	Project       string            `bson:"project" json:"project" mapstructure:"project"`
	User          string            `bson:"user" json:"user" mapstructure:"user"`
	Pipeline      string            `bson:"pipeline" json:"pipeline" mapstructure:"pipeline"` // or workflow
	ClientGroups  string            `bson:"clientgroups" json:"clientgroups" mapstructure:"clientgroups"`
	SubmitTime    time.Time         `bson:"submittime" json:"submittime" mapstructure:"submittime"`
	StartedTime   time.Time         `bson:"startedtime" json:"startedtime" mapstructure:"startedtime"`
	CompletedTime time.Time         `bson:"completedtime" json:"completedtime" mapstructure:"completedtime"`
	Priority      int               `bson:"priority" json:"priority" mapstructure:"priority"`
	Auth          bool              `bson:"auth" json:"auth" mapstructure:"auth"`
	DataToken     string            `bson:"datatoken" json:"-" mapstructure:"-"`
	NoRetry       bool              `bson:"noretry" json:"noretry" mapstructure:"noretry"`
	UserAttr      map[string]string `bson:"userattr" json:"userattr" mapstructure:"userattr"`
	Description   string            `bson:"description" json:"description" mapstructure:"description"`
	Tracking      bool              `bson:"tracking" json:"tracking" mapstructure:"tracking"`
}

func NewInfo() *Info {
	return &Info{
		SubmitTime: time.Now(),
		Priority:   conf.BasePriority,
	}
}

func (this *Info) SetStartedTime(jobid string, t time.Time) (err error) {

	err = DbUpdateJobField(jobid, "info.startedtime", t)
	if err != nil {
		return
	}
	this.StartedTime = t
	return
}
