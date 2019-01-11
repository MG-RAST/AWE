package core

import (
	"fmt"
	"time"

	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/db"
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

	//fmt.Println("(dbUpdateTaskFields) update_value:")
	//spew.Dump(update_value)

	err = c.Update(selector, update_op)
	if err != nil {
		err = fmt.Errorf("(dbUpdateJobTaskFields) (db: %s) Error updating task %s: %s", database, task_id, err.Error())
		return
	}
	return
}

func dbUpdateTaskTime(job_id string, workflow_instance string, task_id string, fieldname string, value time.Time) (err error) {
	update_value := bson.M{"tasks.$." + fieldname: value}
	err = dbUpdateTaskFields(job_id, workflow_instance, task_id, update_value)
	if err != nil {
		err = fmt.Errorf("(dbUpdateTaskTime) dbUpdateTaskFields returned: %s", err.Error())
	}
	return
}

func dbUpdateTaskString(job_id string, workflow_instance string, task_id string, fieldname string, value string) (err error) {
	update_value := bson.M{"tasks.$." + fieldname: value}
	err = dbUpdateTaskFields(job_id, workflow_instance, task_id, update_value)

	if err != nil {
		err = fmt.Errorf("(dbUpdateJobTaskString) job_id=%s, task_id=%s, fieldname=%s, value=%s dbUpdateJobTaskFields returned: %s", job_id, task_id, fieldname, value, err.Error())
	}
	return
}

func dbUpdateTaskBoolean(job_id string, workflow_instance string, task_id string, fieldname string, value bool) (err error) {
	update_value := bson.M{"tasks.$." + fieldname: value}
	err = dbUpdateTaskFields(job_id, workflow_instance, task_id, update_value)
	if err != nil {
		err = fmt.Errorf("(dbUpdateTaskBoolean) dbUpdateJobTaskFields returned: %s", err.Error())
	}
	return

}

func dbUpdateTaskIO(job_id string, workflow_instance string, task_id string, fieldname string, value []*IO) (err error) {
	update_value := bson.M{"tasks.$." + fieldname: value}
	err = dbUpdateTaskFields(job_id, workflow_instance, task_id, update_value)
	if err != nil {
		err = fmt.Errorf("(dbUpdateJobTaskIO) dbUpdateJobTaskFields returned: %s", err.Error())
	}
	return
}

func dbUpdateTaskField(job_id string, workflow_instance string, task_id string, fieldname string, value interface{}) (err error) {
	update_value := bson.M{"tasks.$." + fieldname: value}
	err = dbUpdateTaskFields(job_id, workflow_instance, task_id, update_value)
	if err != nil {
		err = fmt.Errorf("(dbUpdateTaskField) dbUpdateTaskFields returned: %s", err.Error())
	}
	return

}

func dbUpdateTaskInt(job_id string, workflow_instance string, task_id string, fieldname string, value int) (err error) {
	update_value := bson.M{"tasks.$." + fieldname: value}
	err = dbUpdateTaskFields(job_id, workflow_instance, task_id, update_value)
	if err != nil {
		err = fmt.Errorf("(dbUpdateTaskInt) dbUpdateTaskFields returned: %s", err.Error())
	}
	return

}

// func dbUpdateTaskFields(job_id string, workflow_instance_id string, task_id string, update_value bson.M) (err error) {
// 	session := db.Connection.Session.Copy()
// 	defer session.Close()

// 	database := conf.DB_COLL_JOBS
// 	if workflow_instance_id != "" {
// 		database = conf.DB_COLL_SUBWORKFLOWS
// 	}

// 	c := session.DB(conf.MONGODB_DATABASE).C(database)

// 	var selector bson.M
// 	var update_op bson.M

// 	unique_id := job_id + workflow_instance_id
// 	selector = bson.M{"_id": unique_id, "tasks.taskid": task_id}
// 	update_op = bson.M{"$set": update_value}

// 	fmt.Println("(dbUpdateTaskFields) update_value:")
// 	spew.Dump(update_value)

// 	err = c.Update(selector, update_op)
// 	if err != nil {
// 		err = fmt.Errorf("(dbUpdateTaskFields) (db: %s) Error updating task: %s", database, err.Error())
// 		return
// 	}
// 	return
// }
