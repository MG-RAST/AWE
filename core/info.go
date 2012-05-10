package core

import (
	"crypto/md5"
	"fmt"
	"github.com/MG-RAST/AWE/conf"
	"math/rand"
	"time"
)

type Info struct {
	Id       string `bson:"id" json:"id"`
	Name     string `bson:"name" json:"name"`
	Owner    string `bson:"owner" json:"owner"`
	State    string `bson:"state" json:"state"`
	Priority int    `bson:"priority" json:"priority"`
}

func NewInfo() *Info {
	return &Info{Id: id(), State: "Not Started", Priority: conf.BasePriority}
}

func id() string {
	var s []byte
	h := md5.New()
	h.Write([]byte(fmt.Sprint(time.Now().String(), rand.Float64())))
	s = h.Sum(s)
	return fmt.Sprintf("%x", s)
}
