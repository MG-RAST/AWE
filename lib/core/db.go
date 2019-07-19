package core

import (
	"fmt"

	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/db"
	"gopkg.in/mgo.v2/bson"
)

// mongodb has hard limit of 16 MB docuemnt size
var DocumentMaxByte = 16777216

// indexed info fields for search

type StructContainer struct {
	Data interface{} `json:"data"`
}

type DefaultQueryOptions struct {
	Limit  int
	Offset int
	Sort   []string // Order or sortby
}

func dbDelete(q bson.M, coll string) (err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C(coll)
	_, err = c.RemoveAll(q)
	return
}

func dbUpsert(t interface{}) (err error) {
	// test that document not to large
	var nbson []byte
	if nbson, err = bson.Marshal(t); err == nil {
		if len(nbson) >= DocumentMaxByte {
			err = fmt.Errorf("bson document size is greater than limit of %d bytes", DocumentMaxByte)
			return
		}
	}
	session := db.Connection.Session.Copy()
	defer session.Close()
	switch t := t.(type) {
	case *Job:
		c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)
		_, err = c.Upsert(bson.M{"id": t.ID}, &t)
	case *WorkflowInstance:

		err = fmt.Errorf("(dbUpsert) not supported, please use dbInsert or dbUpdate")
		return

	case *JobPerf:
		c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_PERF)
		_, err = c.Upsert(bson.M{"id": t.Id}, &t)
	case *ClientGroup:
		c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_CGS)
		_, err = c.Upsert(bson.M{"id": t.ID}, &t)
	default:
		fmt.Printf("invalid database entry type\n")
	}
	return
}

func dbInsert(t interface{}) (err error) {
	// test that document not to large
	var nbson []byte
	if nbson, err = bson.Marshal(t); err == nil {
		if len(nbson) >= DocumentMaxByte {
			err = fmt.Errorf("bson document size is greater than limit of %d bytes", DocumentMaxByte)
			return
		}
	}
	session := db.Connection.Session.Copy()
	defer session.Close()
	switch t := t.(type) {

	case *WorkflowInstance:
		c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_SUBWORKFLOWS)
		//var info *mgo.ChangeInfo

		//id, _ := t.GetID(false)

		err = c.Insert(&t)
		if err != nil {
			err = fmt.Errorf("(dbUpsert) c.Upsert returned: %s", err.Error())
			return
		}
		//fmt.Println("(dbUpsert) Upsert: " + id)

		//info, err = c.Upsert(bson.M{"id": t.Id}, &t)

		//fmt.Println("dbUpsert: info")
		//spew.Dump(info)

	default:
		fmt.Printf("invalid database entry type\n")
	}
	return
}

func dbUpdate(t interface{}) (err error) {
	// test that document not to large
	var nbson []byte
	if nbson, err = bson.Marshal(t); err == nil {
		if len(nbson) >= DocumentMaxByte {
			err = fmt.Errorf("bson document size is greater than limit of %d bytes", DocumentMaxByte)
			return
		}
	}
	session := db.Connection.Session.Copy()
	defer session.Close()
	switch t := t.(type) {

	case *WorkflowInstance:
		c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_SUBWORKFLOWS)
		//var info *mgo.ChangeInfo

		//id, _ := t.GetID(false)

		err = c.Update(bson.M{"id": t.ID}, &t)
		if err != nil {
			err = fmt.Errorf("(dbUpsert) c.Upsert returned: %s", err.Error())
			return
		}
		//fmt.Println("(dbUpsert) Upsert: " + id)

		//info, err = c.Upsert(bson.M{"id": t.Id}, &t)

		//fmt.Println("dbUpsert: info")
		//spew.Dump(info)

	default:
		fmt.Printf("invalid database entry type\n")
	}
	return
}
