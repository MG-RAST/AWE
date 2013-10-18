// Package db to connect to mongodb
package db

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"labix.org/v2/mgo"
	"time"
)

const (
	DbTimeout = time.Duration(time.Second * 1)
)

var (
	Connection connection
)

type connection struct {
	dbname   string
	username string
	password string
	Session  *mgo.Session
	DB       *mgo.Database
}

func Initialize() (err error) {
	c := connection{}
	s, err := mgo.DialWithTimeout(conf.MONGODB_HOST, DbTimeout)
	if err != nil {
		e := errors.New(fmt.Sprintf("no reachable mongodb server(s) at %s", conf.MONGODB_HOST))
		return e
	}
	c.Session = s
	c.DB = c.Session.DB(conf.MONGODB_DATABASE)
	if conf.MONGODB_USER != "" && conf.MONGODB_PASSWD != "" {
		c.DB.Login(conf.MONGODB_USER, conf.MONGODB_PASSWD)
	}
	Connection = c
	return
}

func Drop() error {
	return Connection.DB.DropDatabase()
}
