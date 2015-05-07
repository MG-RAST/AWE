// Package db to connect to mongodb
package db

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/Shock/shock-server/conf"
	"github.com/MG-RAST/golib/mgo"
	"time"
)

const (
	DbTimeout = time.Duration(time.Second * 1200)
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
	s, err := mgo.DialWithTimeout(conf.MONGODB_HOSTS, DbTimeout)
	if err != nil {
		e := errors.New(fmt.Sprintf("no reachable mongodb server(s) at %s", conf.MONGODB_HOSTS))
		return e
	}
	c.Session = s
	c.DB = c.Session.DB(conf.MONGODB_DATABASE)
	if conf.MONGODB_USER != "" && conf.MONGODB_PASSWORD != "" {
		c.DB.Login(conf.MONGODB_USER, conf.MONGODB_PASSWORD)
	}
	Connection = c
	return
}

func Drop() error {
	return Connection.DB.DropDatabase()
}
