package core

import (
	"fmt"

	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	"github.com/MG-RAST/AWE/lib/db"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// DBGetJobWorkflowInstances _
func DBGetJobWorkflowInstances(q bson.M, options *DefaultQueryOptions, doInit bool) (results []interface{}, count int, err error) {

	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_SUBWORKFLOWS)
	query := c.Find(q)
	count, err = query.Count()
	if err != nil {
		results = nil
		err = fmt.Errorf("(DBGetJobWorkflowInstances) query.Count returned: %s", err.Error())
		return
	}

	if len(options.Sort) > 0 {
		// this allows to provide multiple sort fields
		query = query.Sort(options.Sort...)
	}

	// options.Limit is always defined
	query = query.Limit(options.Limit)

	// options.Offset is always defined
	query = query.Skip(options.Offset)

	//var results []WorkflowInstance
	results = []interface{}{}

	err = query.All(&results)
	if err != nil {
		err = fmt.Errorf("query.All(results) failed: %s", err.Error())
		return
	}
	if results == nil {
		err = fmt.Errorf("(dbFindSort) results == nil")
		return
	}

	// if do_init {
	// 	var count_changed int

	// 	for i, _ := range results {
	// 		var changed bool
	// 		changed, err = results[i].Init(nil)
	// 		if err != nil {
	// 			err = fmt.Errorf("results[i].Init(nil) failed: %s", err.Error())
	// 			return
	// 		}
	// 		if changed {
	// 			count_changed += 1
	// 			results[i].Save(true)
	// 		}
	// 	}
	// 	logger.Debug(1, "%d jobs haven been updated by Init function", count_changed)

	// }

	return
}

func DBGetJobWorkflowInstanceOne(q bson.M, options *DefaultQueryOptions, doInit bool) (result interface{}, err error) {

	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_SUBWORKFLOWS)
	query := c.Find(q)
	// count, err = query.Count()
	// if err != nil {
	// 	result = nil
	// 	err = fmt.Errorf("(DBGetJobWorkflow_instance_One) query.Count returned: %s", err.Error())
	// 	return
	// }

	// if len(options.Sort) > 0 {
	// 	// this allows to provide multiple sort fields
	// 	query = query.Sort(options.Sort...)
	// }

	// options.Limit is always defined
	//query = query.Limit(options.Limit)

	// options.Offset is always defined
	//query = query.Skip(options.Offset)

	//var results []WorkflowInstance
	//result = interface{}

	err = query.One(&result)
	if err != nil {
		err = fmt.Errorf("(DBGetJobWorkflowInstanceOne) query.All(results) failed: %s", err.Error())
		return
	}
	if result == nil {
		err = fmt.Errorf("(DBGetJobWorkflowInstanceOne) result == nil")
		return
	}
	return
}

func dbPushTask(subworkflowIdentifier string, task *Task) (err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_SUBWORKFLOWS)

	//unique_id := jobID + "_" + subworkflow_id
	selector := bson.M{"id": subworkflowIdentifier}

	change := bson.M{"$push": bson.M{"tasks": task}}

	err = c.Update(selector, change)
	if err != nil {
		err = fmt.Errorf("(dbPushTask) Error adding task: " + err.Error())
		return
	}

	return
}

func dbUpdateJobWorkflowInstancesFieldOutputs(subworkflowIdentifier string, outputs cwl.Job_document) (err error) {
	updateValue := bson.M{"outputs": outputs}
	return dbUpdateJobWorkflowInstancesFields(subworkflowIdentifier, updateValue)
}

func dbUpdateJobWorkflowInstancesFieldInt(subworkflowIdentifier string, fieldname string, value int) (err error) {
	updateValue := bson.M{fieldname: value}
	err = dbUpdateJobWorkflowInstancesFields(subworkflowIdentifier, updateValue)
	if err != nil {
		err = fmt.Errorf("(dbUpdateJobWorkflowInstancesFieldInt) (subworkflowIdentifier: %s, fieldname: %s, value: %d) %s", subworkflowIdentifier, fieldname, value, err.Error())
		return
	}
	return
}

func dbUpdateJobWorkflowInstancesFieldString(subworkflowIdentifier string, fieldname string, value string) (err error) {
	updateValue := bson.M{fieldname: value}
	err = dbUpdateJobWorkflowInstancesFields(subworkflowIdentifier, updateValue)
	if err != nil {
		err = fmt.Errorf("(dbUpdateJobWorkflowInstancesFieldString) (subworkflowIdentifier: %s, fieldname: %s, value: %s) %s", subworkflowIdentifier, fieldname, value, err.Error())
		return
	}
	return
}

func dbUpdateJobWorkflowInstancesField(subworkflowIdentifier string, fieldname string, value interface{}) (err error) {
	updateValue := bson.M{fieldname: value}
	return dbUpdateJobWorkflowInstancesFields(subworkflowIdentifier, updateValue)
}

func dbUpdateJobWorkflowInstancesFields(subworkflowIdentifier string, updateValue bson.M) (err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_SUBWORKFLOWS)
	//unique_id := job_id + "_" + subworkflow_id
	selector := bson.M{"id": subworkflowIdentifier}

	err = c.Update(selector, bson.M{"$set": updateValue})
	if err != nil {
		err = fmt.Errorf("(dbUpdateJobWorkflowInstancesFields) Error updating workflow_instance (_id: %s): %s", subworkflowIdentifier, err.Error())
		return
	}
	return
}

func dbIncrementJobWorkflowInstancesField(subworkflowIdentifier string, field string, value int) (err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_SUBWORKFLOWS)
	//selector := bson.M{"id": job_id, "workflow_instances.id": subworkflow_id}
	//unique_id := job_id + "_" + subworkflow_id
	selector := bson.M{"id": subworkflowIdentifier}
	//err = c.Update(selector, bson.M{"$set": updateValue})

	change := mgo.Change{
		Update:    bson.M{"$inc": bson.M{field: value}},
		ReturnNew: true,
	}
	//var info *mgo.ChangeInfo

	_, err = c.Find(selector).Apply(change, nil)
	if err != nil {
		err = fmt.Errorf("(dbIncrementJobWorkflow_instancesField) Error updating workflow_instance %s  (field %s, value: %d): %s", subworkflowIdentifier, field, value, err.Error())
		return
	}

	return
}

// dbUpdateWorkflowInstancesFields _
func dbUpdateWorkflowInstancesFields(subworkflowIdentifier string, updateValue bson.M) (err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_SUBWORKFLOWS)
	//uniqueID := jobID + "_" + subworkflowID
	selector := bson.M{"id": subworkflowIdentifier}

	err = c.Update(selector, bson.M{"$set": updateValue})
	if err != nil {
		err = fmt.Errorf("Error updating job fields: %s", err.Error())
		return
	}
	return
}
