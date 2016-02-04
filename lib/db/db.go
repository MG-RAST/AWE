// Package db to connect to mongodb
package db

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	mgo "github.com/MG-RAST/AWE/vendor/gopkg.in/mgo.v2"
	"time"
)

const (
	DialTimeout  = time.Duration(time.Second * 10)
	DialAttempts = 3
)

var (
	Connection connection
	DbTimeout  = time.Second * time.Duration(conf.MONGODB_TIMEOUT)
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

	// test connection
	canDial := false
	for i := 0; i < DialAttempts; i++ {
		s, err := mgo.DialWithTimeout(conf.MONGODB_HOST, DialTimeout)
		if err == nil {
			s.Close()
			canDial = true
			break
		}
	}
	if !canDial {
		return errors.New(fmt.Sprintf("no reachable mongodb server(s) at %s", conf.MONGODB_HOST))
	}

	// get handle
	s, err := mgo.DialWithTimeout(conf.MONGODB_HOST, DbTimeout)
	if err != nil {
		return errors.New(fmt.Sprintf("no reachable mongodb server(s) at %s", conf.MONGODB_HOST))
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
