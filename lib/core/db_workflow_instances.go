package core

import (
	"fmt"

	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	"github.com/MG-RAST/AWE/lib/db"
	"github.com/davecgh/go-spew/spew"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func dbUpdateJobWorkflow_instancesFieldOutputs(job_id string, subworkflow_id string, outputs cwl.Job_document) (err error) {
	update_value := bson.M{"outputs": outputs}
	return dbUpdateJobWorkflow_instancesFields(job_id, subworkflow_id, update_value)
}

func dbUpdateJobWorkflow_instancesFieldInt(job_id string, subworkflow_id string, fieldname string, value int) (err error) {
	update_value := bson.M{fieldname: value}
	err = dbUpdateJobWorkflow_instancesFields(job_id, subworkflow_id, update_value)
	if err != nil {
		err = fmt.Errorf("(dbUpdateJobWorkflow_instancesFieldInt) (subworkflow_id: %s, fieldname: %s, value: %d) %s", subworkflow_id, fieldname, value, err.Error())
		return
	}
	return
}

func dbUpdateJobWorkflow_instancesField(job_id string, subworkflow_id string, fieldname string, value interface{}) (err error) {
	update_value := bson.M{fieldname: value}
	return dbUpdateJobWorkflow_instancesFields(job_id, subworkflow_id, update_value)
}

func dbUpdateJobWorkflow_instancesFields(job_id string, subworkflow_id string, update_value bson.M) (err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)
	selector := bson.M{"_id": job_id + subworkflow_id}

	err = c.Update(selector, bson.M{"$set": update_value})
	if err != nil {
		err = fmt.Errorf("(dbUpdateJobWorkflow_instancesFields) Error updating workflow_instance: " + err.Error())
		return
	}
	return
}

func dbIncrementJobWorkflow_instancesField(job_id string, subworkflow_id string, field string, value int) (new_value int, err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)
	//selector := bson.M{"id": job_id, "workflow_instances.id": subworkflow_id}
	selector := bson.M{"_id": job_id + subworkflow_id}
	//err = c.Update(selector, bson.M{"$set": update_value})

	var result interface{}

	new_value = -1

	change := mgo.Change{
		Update:    bson.M{"$inc": bson.M{field: value}},
		ReturnNew: true,
	}
	//var info *mgo.ChangeInfo
	_, err = c.Find(selector).Apply(change, result)
	if err != nil {
		err = fmt.Errorf("(dbIncrementJobWorkflow_instancesField) Error updating workflow_instance: " + err.Error())
		return
	}

	spew.Dump(result)

	panic("done")

	return
}
