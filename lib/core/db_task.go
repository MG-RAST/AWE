package core

import (
	"fmt"

	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/db"
	"github.com/davecgh/go-spew/spew"
	"gopkg.in/mgo.v2/bson"
)

func dbUpdateTaskFields(job_id string, workflow_instance_id string, task_id string, update_value bson.M) (err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()

	database := conf.DB_COLL_JOBS
	if workflow_instance_id != "" {
		database = conf.DB_COLL_SUBWORKFLOWS
	}

	c := session.DB(conf.MONGODB_DATABASE).C(database)

	unique_id := job_id + workflow_instance_id
	selector := bson.M{"_id": unique_id, "tasks.taskid": task_id}
	update_op := bson.M{"$set": update_value}

	fmt.Println("(dbUpdateTaskFields) update_value:")
	spew.Dump(update_value)

	err = c.Update(selector, update_op)
	if err != nil {
		err = fmt.Errorf("(dbUpdateJobTaskFields) (db: %s) Error updating task: %s", database, err.Error())
		return
	}
	return
}
