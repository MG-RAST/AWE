package core

import (
	"fmt"
	"time"

	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/db"
	"gopkg.in/mgo.v2/bson"
)

// func dbUpdateTaskFields_DEPRECTAED(jobID string, workflowInstanceID string, taskID string, updateValue bson.M) (err error) {
// 	session := db.Connection.Session.Copy()
// 	defer session.Close()

// 	database := conf.DB_COLL_JOBS
// 	if workflowInstanceID != "" {
// 		database = conf.DB_COLL_SUBWORKFLOWS
// 	}

// 	c := session.DB(conf.MONGODB_DATABASE).C(database)

// 	uniqueID := jobID + "_" + workflowInstanceID
// 	selector := bson.M{"_id": uniqueID, "tasks.taskid": taskID}
// 	updateOp := bson.M{"$set": updateValue}

// 	//fmt.Println("(dbUpdateTaskFields) updateValue:")
// 	//spew.Dump(updateValue)

// 	err = c.Update(selector, updateOp)
// 	if err != nil {
// 		err = fmt.Errorf("(dbUpdateJobTaskFields) (db: %s) Error updating task %s (uniqueID: %s): %s", database, taskID, uniqueID, err.Error())
// 		return
// 	}
// 	return
// }

func dbUpdateTaskFields(workflowInstanceID string, taskID string, updateValue bson.M) (err error) {

	if workflowInstanceID == "" {
		err = fmt.Errorf("(dbUpdateWITaskFields) workflowInstanceID empty ! ")
		return
	}

	session := db.Connection.Session.Copy()
	defer session.Close()

	database := conf.DB_COLL_SUBWORKFLOWS

	c := session.DB(conf.MONGODB_DATABASE).C(database)

	selector := bson.M{"id": workflowInstanceID, "tasks.taskid": taskID}
	updateOp := bson.M{"$set": updateValue}

	//fmt.Println("(dbUpdateTaskFields) updateValue:")
	//spew.Dump(updateValue)

	err = c.Update(selector, updateOp)
	if err != nil {
		err = fmt.Errorf("(dbUpdateWITaskFields) (db: %s) Error updating task %s (workflowInstanceID: %s): %s", database, taskID, workflowInstanceID, err.Error())
		return
	}
	return
}

func dbUpdateTaskTime(workflowInstanceID string, taskID string, fieldname string, value time.Time) (err error) {
	updateValue := bson.M{"tasks.$." + fieldname: value}
	err = dbUpdateTaskFields(workflowInstanceID, taskID, updateValue)
	if err != nil {
		err = fmt.Errorf("(dbUpdateTaskTime) dbUpdateTaskFields returned: %s", err.Error())
	}
	return
}

func dbUpdateTaskString(workflowInstanceID string, taskID string, fieldname string, value string) (err error) {
	updateValue := bson.M{"tasks.$." + fieldname: value}
	err = dbUpdateTaskFields(workflowInstanceID, taskID, updateValue)

	if err != nil {
		err = fmt.Errorf("(dbUpdateJobTaskString) workflowInstanceID=%s, taskID=%s, fieldname=%s, value=%s dbUpdateJobTaskFields returned: %s", workflowInstanceID, taskID, fieldname, value, err.Error())
	}
	return
}

func dbUpdateTaskBoolean(workflowInstanceID string, taskID string, fieldname string, value bool) (err error) {
	updateValue := bson.M{"tasks.$." + fieldname: value}
	err = dbUpdateTaskFields(workflowInstanceID, taskID, updateValue)
	if err != nil {
		err = fmt.Errorf("(dbUpdateTaskBoolean) dbUpdateJobTaskFields returned: %s", err.Error())
	}
	return

}

func dbUpdateTaskIO(workflowInstanceID string, taskID string, fieldname string, value []*IO) (err error) {
	updateValue := bson.M{"tasks.$." + fieldname: value}
	err = dbUpdateTaskFields(workflowInstanceID, taskID, updateValue)
	if err != nil {
		err = fmt.Errorf("(dbUpdateJobTaskIO) dbUpdateJobTaskFields returned: %s", err.Error())
	}
	return
}

func dbUpdateTaskField(workflowInstanceID, taskID string, fieldname string, value interface{}) (err error) {
	updateValue := bson.M{"tasks.$." + fieldname: value}
	err = dbUpdateTaskFields(workflowInstanceID, taskID, updateValue)
	if err != nil {
		err = fmt.Errorf("(dbUpdateTaskField) dbUpdateTaskFields returned: %s", err.Error())
	}
	return

}

func dbUpdateTaskInt(workflowInstanceID, taskID string, fieldname string, value int) (err error) {
	updateValue := bson.M{"tasks.$." + fieldname: value}
	err = dbUpdateTaskFields(workflowInstanceID, taskID, updateValue)
	if err != nil {
		err = fmt.Errorf("(dbUpdateTaskInt) dbUpdateTaskFields returned: %s", err.Error())
	}
	return

}

// func dbUpdateTaskFields(jobID string, workflow_instance_id string, taskID string, updateValue bson.M) (err error) {
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
// 	update_op = bson.M{"$set": updateValue}

// 	fmt.Println("(dbUpdateTaskFields) updateValue:")
// 	spew.Dump(updateValue)

// 	err = c.Update(selector, update_op)
// 	if err != nil {
// 		err = fmt.Errorf("(dbUpdateTaskFields) (db: %s) Error updating task: %s", database, err.Error())
// 		return
// 	}
// 	return
// }
