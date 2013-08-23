package core

import (
	//	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/conf"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"os"
	"time"
)

const (
	DbTimeout = time.Duration(time.Second * 1)
)

type db struct {
	Jobs    *mgo.Collection
	Session *mgo.Session
}

func InitDB() {
	d, err := DBConnect()
	defer d.Close()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR: no reachable mongodb servers")
		os.Exit(1)
	}
	idIdx := mgo.Index{Key: []string{"id"}, Unique: true}
	err = d.Jobs.EnsureIndex(idIdx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: fatal mongodb initialization error: %v", err)
		os.Exit(1)
	}
}

func DBConnect() (d *db, err error) {
	session, err := mgo.DialWithTimeout(conf.MONGODB, DbTimeout)
	if err != nil {
		return
	}
	d = &db{Jobs: session.DB("AWEDB").C("Jobs"), Session: session}
	return
}

func DropDB() (err error) {
	d, err := DBConnect()
	defer d.Close()
	if err != nil {
		return err
	}
	return d.Jobs.DropCollection()
}

func (d *db) Upsert(job *Job) (err error) {
	_, err = d.Jobs.Upsert(bson.M{"id": job.Id}, &job)
	return
}

func (d *db) FindById(id string, result *Job) (err error) {
	err = d.Jobs.Find(bson.M{"id": id}).One(&result)
	return
}

func (d *db) FindJobs(ids []string, results *[]*Job) (err error) {
	err = d.Jobs.Find(bson.M{"id": bson.M{"$in": ids}}).All(results)
	return
}

func (d *db) GetAll(q bson.M, results *Jobs) (err error) {
	err = d.Jobs.Find(q).All(results)
	return
}

func (d *db) GetAllLimitOffset(q bson.M, results *Jobs, limit int, offset int) (err error) {
	err = d.Jobs.Find(q).Limit(limit).Skip(offset).All(results)
	return
}

func (d *db) GetAllRecent(q bson.M, results *Jobs, recent int) (err error) {
	err = d.Jobs.Find(q).Sort("-jid").Limit(recent).All(results)
	return
}

func (d *db) Close() {
	d.Session.Close()
	return
}
