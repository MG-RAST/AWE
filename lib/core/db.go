package core

import (
	"errors"
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
	if nbson, err := bson.Marshal(t); err == nil {
		if len(nbson) >= DocumentMaxByte {
			return errors.New(fmt.Sprintf("bson document size is greater than limit of %d bytes", DocumentMaxByte))
		}
	}
	session := db.Connection.Session.Copy()
	defer session.Close()
	switch t := t.(type) {
	case *Job:
		c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)
		_, err = c.Upsert(bson.M{"id": t.Id}, &t)
	case *WorkflowInstance:
		c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_SUBWORKFLOWS)
		//var info *mgo.ChangeInfo

		id, _ := t.GetId(false)
		_, err = c.Upsert(bson.M{"_id": id}, &t)

		if err != nil {
			err = fmt.Errorf("(dbUpsert) c.Upsert returned: %s", err.Error())
			return
		}
		fmt.Println("(dbUpsert) Upsert: " + id)

		//info, err = c.Upsert(bson.M{"id": t.Id}, &t)

		//fmt.Println("dbUpsert: info")
		//spew.Dump(info)

	case *JobPerf:
		c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_PERF)
		_, err = c.Upsert(bson.M{"id": t.Id}, &t)
	case *ClientGroup:
		c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_CGS)
		_, err = c.Upsert(bson.M{"id": t.Id}, &t)
	default:
		fmt.Printf("invalid database entry type\n")
	}
	return
}
