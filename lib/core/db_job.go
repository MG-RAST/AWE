package core

import (
	"errors"
	"fmt"
	"time"

	"github.com/MG-RAST/AWE/lib/acl"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/db"
	"github.com/MG-RAST/AWE/lib/logger"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// JobInfoIndexes _
var JobInfoIndexes = []string{"name", "submittime", "completedtime", "pipeline", "clientgroups", "project", "service", "user", "priority", "userattr.submission"}

// HasInfoField _
func HasInfoField(a string) bool {
	for _, b := range JobInfoIndexes {
		if b == a {
			return true
		}
	}
	return false
}

// InitJobDB _
func InitJobDB() {
	session := db.Connection.Session.Copy()
	defer session.Close()
	cj := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)
	cj.EnsureIndex(mgo.Index{Key: []string{"acl.owner"}, Background: true})
	cj.EnsureIndex(mgo.Index{Key: []string{"acl.read"}, Background: true})
	cj.EnsureIndex(mgo.Index{Key: []string{"acl.write"}, Background: true})
	cj.EnsureIndex(mgo.Index{Key: []string{"acl.delete"}, Background: true})
	cj.EnsureIndex(mgo.Index{Key: []string{"id"}, Unique: true})
	cj.EnsureIndex(mgo.Index{Key: []string{"state"}, Background: true})
	cj.EnsureIndex(mgo.Index{Key: []string{"expiration"}, Background: true})
	cj.EnsureIndex(mgo.Index{Key: []string{"updatetime"}, Background: true})
	for _, v := range JobInfoIndexes {
		cj.EnsureIndex(mgo.Index{Key: []string{"info." + v}, Background: true})
	}
	cp := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_PERF)
	cp.EnsureIndex(mgo.Index{Key: []string{"id"}, Unique: true})

	cw := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_SUBWORKFLOWS)
	//cw.EnsureIndex(mgo.Index{Key: []string{"id"}, Unique: true}) not needed, already got _id
	cw.EnsureIndex(mgo.Index{Key: []string{"job_id"}, Unique: false})
}

func dbCount(q bson.M) (count int, err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)
	if count, err = c.Find(q).Count(); err != nil {
		return 0, err
	}

	return count, nil

}

// get a minimal subset of the job documents required for an admin overview
// for all completed jobs younger than a month and all running jobs
func dbAdminData(special string) (data []interface{}, err error) {
	// get a DB connection
	session := db.Connection.Session.Copy()

	// close the connection when the function completes
	defer session.Close()

	// set the database and collection
	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)

	// get the completed jobs that have a completed time not older than one month
	var completedjobs = bson.M{"state": "completed", "info.completedtime": bson.M{"$gt": time.Now().AddDate(0, -1, 0)}}

	// get all runnning jobs (those not deleted and not completed)
	var runningjobs = bson.M{"state": bson.M{"$nin": []string{"completed", "deleted"}}}

	// select only those fields required for the output
	var resultfields = bson.M{"_id": 0, "state": 1, "info.name": 1, "info.submittime": 1, "info.startedtime": 1, "info.completedtime": 1, "info.pipeline": 1, "tasks.createdDate": 1, "tasks.startedDate": 1, "tasks.completedDate": 1, "tasks.state": 1, "tasks.inputs.size": 1, "tasks.outputs.size": 1, special: 1}

	// return all data without iterating
	err = c.Find(bson.M{"$or": []bson.M{completedjobs, runningjobs}}).Select(resultfields).All(&data)

	return
}

func dbFindSort(filterQuery bson.M, results *Jobs, options map[string]int, sortby string, doInit bool) (count int, err error) {
	if sortby == "" {
		return 0, errors.New("sortby must be an nonempty string")
	}
	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)

	query := c.Find(filterQuery)

	if count, err = query.Count(); err != nil {
		return 0, err
	}

	if limit, has := options["limit"]; has {
		if offset, has := options["offset"]; has {
			err = query.Sort(sortby).Limit(limit).Skip(offset).All(results)
			if results != nil {
				if doInit {
					_, err = results.Init()
					if err != nil {
						return
					}
				}
			}
			return
		}
	}
	err = query.Sort(sortby).All(results)
	if err != nil {
		err = fmt.Errorf("query.Sort(sortby).All(results) failed: %s", err.Error())
		return
	}
	if results == nil {
		err = fmt.Errorf("(dbFindSort) results == nil")
		return
	}

	var countChanged int

	if doInit {
		countChanged, err = results.Init()
		if err != nil {
			err = fmt.Errorf("(dbFindSort) results.Init() failed: %s", err.Error())
			return
		}
		logger.Debug(1, "%d jobs haven been updated by Init function", countChanged)
	}
	return
}

// DbFindDistinct _
func DbFindDistinct(q bson.M, d string) (results interface{}, err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)
	err = c.Find(q).Distinct("info."+d, &results)
	return
}

func dbUpdateJobState(jobID string, newState string, notes string) (err error) {
	logger.Debug(3, "(dbUpdateJobState) jobID: %s", jobID)
	var updateValue bson.M

	if newState == JOB_STAT_COMPLETED {
		updateValue = bson.M{"state": newState, "notes": notes, "info.completedtime": time.Now()}
	} else {
		updateValue = bson.M{"state": newState, "notes": notes}
	}
	return dbUpdateJobFields(jobID, updateValue)
}

func dbGetJobTasks(jobID string) (tasks []*Task, err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)

	selector := bson.M{"id": jobID}
	fieldname := "tasks"
	err = c.Find(selector).Select(bson.M{fieldname: 1}).All(&tasks)
	if err != nil {
		err = fmt.Errorf("(dbGetJobTasks) Error getting tasks from jobID %s: %s", jobID, err.Error())
		return
	}
	return
}

func dbGetJobTaskString(jobID string, taskID string, fieldname string) (result string, err error) {
	var cont StructContainer
	err = dbGetJobTaskField(jobID, taskID, fieldname, &cont)
	if err != nil {
		return
	}
	result = cont.Data.(string)
	return
}

func dbGetJobTaskTime(jobID string, taskID string, fieldname string) (result time.Time, err error) {
	var cont StructContainer
	err = dbGetJobTaskField(jobID, taskID, fieldname, &cont)
	if err != nil {
		return
	}
	result = cont.Data.(time.Time)
	return
}

func dbGetJobTaskInt(jobID string, taskID string, fieldname string) (result int, err error) {
	var cont StructContainer
	err = dbGetJobTaskField(jobID, taskID, fieldname, &cont)
	if err != nil {
		return
	}
	result = cont.Data.(int)
	return
}

func dbFind(q bson.M, results *Jobs, options map[string]int) (count int, err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)
	query := c.Find(q)
	if count, err = query.Count(); err != nil {
		return 0, err
	}
	if limit, has := options["limit"]; has {
		if offset, has := options["offset"]; has {
			err = query.Limit(limit).Skip(offset).All(results)
			if results != nil {
				_, err = results.Init()
				if err != nil {
					return
				}
			}
			return
		}

		return 0, errors.New("store.db.Find options limit and offset must be used together")

	}
	err = query.All(results)
	if results != nil {
		_, err = results.Init()
		if err != nil {
			return
		}
	}
	return
}

func dbUpdateJobFields(jobID string, updateValue bson.M) (err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)
	selector := bson.M{"id": jobID}

	err = c.Update(selector, bson.M{"$set": updateValue})
	if err != nil {
		err = fmt.Errorf("Error updating job fields: " + err.Error())
		return
	}
	return
}

func dbUpdateJobFieldBoolean(jobID string, fieldname string, value bool) (err error) {
	updateValue := bson.M{fieldname: value}
	return dbUpdateJobFields(jobID, updateValue)
}

func dbUpdateJobFieldString(jobID string, fieldname string, value string) (err error) {
	updateValue := bson.M{fieldname: value}
	return dbUpdateJobFields(jobID, updateValue)
}

func dbUpdateJobFieldInt(jobID string, fieldname string, value int) (err error) {
	updateValue := bson.M{fieldname: value}
	return dbUpdateJobFields(jobID, updateValue)
}

func dbUpdateJobFieldTime(jobID string, fieldname string, value time.Time) (err error) {
	updateValue := bson.M{fieldname: value}
	return dbUpdateJobFields(jobID, updateValue)
}

func dbUpdateJobFieldNull(jobID string, fieldname string) (err error) {
	updateValue := bson.M{fieldname: nil}
	return dbUpdateJobFields(jobID, updateValue)
}

func dbUpdateJobTaskFields(jobID string, workflowInstanceID string, taskID string, updateValue bson.M) (err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()

	database := conf.DB_COLL_JOBS
	if workflowInstanceID != "" {
		database = conf.DB_COLL_SUBWORKFLOWS
	}

	c := session.DB(conf.MONGODB_DATABASE).C(database)

	var selector bson.M
	var updateOp bson.M
	if workflowInstanceID == "" {
		selector = bson.M{"id": jobID, "tasks.taskid": taskID}
		updateOp = bson.M{"$set": updateValue}
	} else {
		err = fmt.Errorf("not supported")
		return

		//unique_id := jobID + workflowInstance_id
		//selector = bson.M{"_id": unique_id, "tasks.taskid": taskID}
		//updateOp = bson.M{"$set": updateValue}
	}

	//fmt.Println("(dbUpdateJobTaskFields) updateValue:")
	//spew.Dump(updateValue)

	err = c.Update(selector, updateOp)
	if err != nil {
		err = fmt.Errorf("(dbUpdateJobTaskFields) (db: %s) Error updating task %s: %s", database, taskID, err.Error())
		return
	}
	return
}

func dbUpdateJobTaskField(jobID string, workflowInstance string, taskID string, fieldname string, value interface{}) (err error) {
	updateValue := bson.M{"tasks.$." + fieldname: value}
	err = dbUpdateJobTaskFields(jobID, workflowInstance, taskID, updateValue)
	if err != nil {
		err = fmt.Errorf("(dbUpdateJobTaskField) dbUpdateJobTaskFields returned: %s", err.Error())
	}
	return

}

func dbUpdateJobTaskInt(jobID string, workflowInstance string, taskID string, fieldname string, value int) (err error) {
	updateValue := bson.M{"tasks.$." + fieldname: value}
	err = dbUpdateJobTaskFields(jobID, workflowInstance, taskID, updateValue)
	if err != nil {
		err = fmt.Errorf("(dbUpdateJobTaskInt) dbUpdateJobTaskFields returned: %s", err.Error())
	}
	return

}
func dbUpdateJobTaskBoolean(jobID string, workflowInstance string, taskID string, fieldname string, value bool) (err error) {
	updateValue := bson.M{"tasks.$." + fieldname: value}
	err = dbUpdateJobTaskFields(jobID, workflowInstance, taskID, updateValue)
	if err != nil {
		err = fmt.Errorf("(dbUpdateJobTaskBoolean) dbUpdateJobTaskFields returned: %s", err.Error())
	}
	return

}

func dbUpdateJobTaskString(jobID string, workflowInstance string, taskID string, fieldname string, value string) (err error) {
	updateValue := bson.M{"tasks.$." + fieldname: value}
	err = dbUpdateJobTaskFields(jobID, workflowInstance, taskID, updateValue)

	if err != nil {
		err = fmt.Errorf("(dbUpdateJobTaskString) jobID=%s, taskID=%s, fieldname=%s, value=%s dbUpdateJobTaskFields returned: %s", jobID, taskID, fieldname, value, err.Error())
	}
	return
}

func dbUpdateJobTaskTime(jobID string, workflowInstance string, taskID string, fieldname string, value time.Time) (err error) {
	updateValue := bson.M{"tasks.$." + fieldname: value}
	err = dbUpdateJobTaskFields(jobID, workflowInstance, taskID, updateValue)
	if err != nil {
		err = fmt.Errorf("(dbUpdateJobTaskTime) (fieldname %s) dbUpdateJobTaskFields returned: %s", fieldname, err.Error())
	}
	return
}

func dbUpdateJobTaskPartition(jobID string, workflowInstance string, taskID string, partition *PartInfo) (err error) {
	updateValue := bson.M{"tasks.$.partinfo": partition}
	err = dbUpdateJobTaskFields(jobID, workflowInstance, taskID, updateValue)
	if err != nil {
		err = fmt.Errorf("(dbUpdateJobTaskPartition) dbUpdateJobTaskFields returned: %s", err.Error())
	}
	return
}

func dbUpdateJobTaskIO(jobID string, workflowInstance string, taskID string, fieldname string, value []*IO) (err error) {
	updateValue := bson.M{"tasks.$." + fieldname: value}
	err = dbUpdateJobTaskFields(jobID, workflowInstance, taskID, updateValue)
	if err != nil {
		err = fmt.Errorf("(dbUpdateJobTaskIO) dbUpdateJobTaskFields returned: %s", err.Error())
	}
	return
}

func dbIncrementJobTaskField(jobID string, taskID string, fieldname string, incrementValue int) (err error) {

	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)

	selector := bson.M{"id": jobID, "tasks.taskid": taskID}

	updateValue := bson.M{"tasks.$." + fieldname: incrementValue}

	err = c.Update(selector, bson.M{"$inc": updateValue})
	if err != nil {
		err = fmt.Errorf("Error incrementing jobID=%s fieldname=%s by %d: %s", jobID, fieldname, incrementValue, err.Error())
		return
	}

	return
}

// DbUpdateJobField _
func DbUpdateJobField(jobID string, key string, value interface{}) (err error) {

	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)

	query := bson.M{"id": jobID}
	updateValue := bson.M{key: value}
	update := bson.M{"$set": updateValue}

	err = c.Update(query, update)
	if err != nil {
		err = fmt.Errorf("Error updating job %s and key %s in job: %s", jobID, key, err.Error())
		return
	}

	return
}

// LoadJob _
func LoadJob(id string) (job *Job, err error) {
	job = NewJob()
	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)

	// A) get job document
	err = c.Find(bson.M{"id": id}).One(&job)
	if err != nil {
		job = nil
		err = fmt.Errorf("(LoadJob) c.Find failed: %s", err.Error())
		return
	}

	// continue A, initialize
	var jobChanged bool
	jobChanged, err = job.Init() // values have already been set at this point...
	if err != nil {
		err = fmt.Errorf("(LoadJob) job.Init failed: %s", err.Error())
		return
	}

	// B) get WorkflowInstances
	if job.IsCWL {

		c2 := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_SUBWORKFLOWS)

		WIsIf := []interface{}{} // have to use interface, because mongo cannot handle interface types

		err = c2.Find(bson.M{"job_id": id}).All(&WIsIf)
		if err != nil {
			job = nil
			err = fmt.Errorf("(LoadJob) (DB_COLL_SUBWORKFLOWS) c.Find failed: %s", err.Error())
			return
		}

		if len(WIsIf) == 0 {
			err = fmt.Errorf("(LoadJob) no matching WorkflowInstances found for job %s", id)
			return
		}

		var wis []*WorkflowInstance
		wis, err = NewWorkflowInstanceArrayFromInterface(WIsIf, job, job.WorkflowContext)
		if err != nil {
			err = fmt.Errorf("(LoadJob) NewWorkflowInstanceArrayFromInterface returned: %s", err.Error())
			return
		}

		if len(wis) == 0 {
			err = fmt.Errorf("(LoadJob) NewWorkflowInstanceArrayFromInterface returned no WorkflowInstances for job %s", id)
			return
		}

		for i := range wis {
			var wiChanged bool
			wi := wis[i]
			wiChanged, err = wi.Init(job)
			if err != nil {
				err = fmt.Errorf("(LoadJob) wis[i].Init returned: %s", err.Error())
				return
			}
			if wiChanged {
				err = wi.Save(true)
				if err != nil {
					err = fmt.Errorf("(LoadJob) wi.Save() returned: %s", err.Error())
					return
				}
			}
			// add WorkflowInstance to job

			fmt.Printf("(LoadJob) loading: %s\n", wi.LocalID)

			err = job.AddWorkflowInstance(wi, DbSyncFalse, true) // load from database
			if err != nil {
				err = fmt.Errorf("(LoadJob) AddWorkflowInstance returned: %s", err.Error())
				return
			}

			wiUniqueID, _ := wi.GetID(true)

			err = GlobalWorkflowInstanceMap.Add(wiUniqueID, wi)
			if err != nil {
				err = fmt.Errorf("(LoadJob) GlobalWorkflowInstanceMap.Add returned: %s", err.Error())
				return
			}

		}
	}

	// To fix incomplete or inconsistent database entries
	if jobChanged {
		err = job.Save()
		if err != nil {
			err = fmt.Errorf("(LoadJob) job.Save() returned: %s", err.Error())
			return
		}
	}

	return

}

// LoadJobPerf _
func LoadJobPerf(id string) (perf *JobPerf, err error) {
	perf = new(JobPerf)
	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_PERF)
	if err = c.Find(bson.M{"id": id}).One(&perf); err == nil {
		return perf, nil
	}
	return nil, err
}

func dbGetJobTaskField(jobID string, taskID string, fieldname string, result *StructContainer) (err error) {

	arrayName := "tasks"
	idField := "taskid"
	return dbGetJobArrayField(jobID, taskID, arrayName, idField, fieldname, result)
}

// TODO: warning: this does not cope with subfields such as "partinfo.index"
func dbGetJobArrayField(jobID string, taskID string, arrayName string, idField string, fieldname string, result *StructContainer) (err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)
	selector := bson.M{"id": jobID}

	projection := bson.M{arrayName: bson.M{"$elemMatch": bson.M{idField: taskID}}, arrayName + "." + fieldname: 1}
	tempResult := bson.M{}

	err = c.Find(selector).Select(projection).One(&tempResult)
	if err != nil {
		err = fmt.Errorf("(dbGetJobArrayField) Error getting field from jobID %s , %s=%s and fieldname %s: %s", jobID, idField, taskID, fieldname, err.Error())
		return
	}

	//logger.Debug(3, "GOT: %v", temp_result)

	tasksUnknown := tempResult[arrayName]
	tasks, ok := tasksUnknown.([]interface{})

	if !ok {
		err = fmt.Errorf("(dbGetJobArrayField) Array expected, but not found")
		return
	}
	if len(tasks) == 0 {
		err = fmt.Errorf("(dbGetJobArrayField) result task array empty")
		return
	}

	firstTask := tasks[0].(bson.M)
	testResult, ok := firstTask[fieldname]

	if !ok {
		err = fmt.Errorf("(dbGetJobArrayField) Field %s not in task object", fieldname)
		return
	}
	result.Data = testResult

	//logger.Debug(3, "GOT2: %v", result)
	logger.Debug(3, "(dbGetJobArrayField) %s got something", fieldname)
	return
}

func dbGetJobTask(jobID string, taskID string) (result *Task, err error) {
	dummyJob := NewJob()
	dummyJob.Init()

	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)

	selector := bson.M{"id": jobID}
	projection := bson.M{"tasks": bson.M{"$elemMatch": bson.M{"taskid": taskID}}}

	err = c.Find(selector).Select(projection).One(&dummyJob)
	if err != nil {
		err = fmt.Errorf("(dbGetJobTask) Error getting field from jobID %s , taskID %s : %s", jobID, taskID, err.Error())
		return
	}
	if len(dummyJob.Tasks) != 1 {
		err = fmt.Errorf("(dbGetJobTask) len(dummy_job.Tasks) != 1   len(dummy_job.Tasks)=%d", len(dummyJob.Tasks))
		return
	}

	result = dummyJob.Tasks[0]
	return
}

func dbGetPrivateEnv(jobID string, taskID string) (result map[string]string, err error) {
	task, err := dbGetJobTask(jobID, taskID)
	if err != nil {
		return
	}
	result = task.Cmd.Environ.Private
	for key, val := range result {
		logger.Debug(3, "(dbGetPrivateEnv) got %s=%s ", key, val)
	}
	return
}

func dbGetJobFieldInt(jobID string, fieldname string) (result int, err error) {
	err = dbGetJobField(jobID, fieldname, &result)
	if err != nil {
		return
	}
	return
}

func dbGetJobFieldString(jobID string, fieldname string) (result string, err error) {
	err = dbGetJobField(jobID, fieldname, &result)
	logger.Debug(3, "(dbGetJobFieldString) result: %s", result)
	if err != nil {
		return
	}
	return
}

func dbGetJobField(jobID string, fieldname string, result interface{}) (err error) {

	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)
	selector := bson.M{"id": jobID}

	err = c.Find(selector).Select(bson.M{fieldname: 1}).One(&result)
	if err != nil {
		err = fmt.Errorf("(dbGetJobField) Error getting field from jobID %s and fieldname %s: %s", jobID, fieldname, err.Error())
		return
	}
	return
}

// JobACL _
type JobACL struct {
	ACL acl.Acl `bson:"acl" json:"-"`
}

// DBGetJobACL _
func DBGetJobACL(jobID string) (_acl acl.Acl, err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)
	selector := bson.M{"id": jobID}

	job := JobACL{}

	err = c.Find(selector).Select(bson.M{"acl": 1}).One(&job)
	if err != nil {
		err = fmt.Errorf("Error getting acl field from jobID %s: %s", jobID, err.Error())
		return
	}

	_acl = job.ACL
	return
}

func dbGetJobFieldTime(jobID string, fieldname string) (result time.Time, err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)
	selector := bson.M{"id": jobID}

	err = c.Find(selector).Select(bson.M{fieldname: 1}).One(&result)
	if err != nil {
		err = fmt.Errorf("(dbGetJobFieldTime) Error getting field from jobID %s and fieldname %s: %s", jobID, fieldname, err.Error())
		return
	}
	return
}

func dbPushJobTask(jobID string, task *Task) (err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)

	selector := bson.M{"id": jobID}
	change := bson.M{"$push": bson.M{"tasks": task}}

	err = c.Update(selector, change)
	if err != nil {
		err = fmt.Errorf("Error adding task: " + err.Error())
		return
	}

	//fmt.Printf("(dbPushJobTask) added task: %s\n", task.Task_Unique_Identifier.TaskName)
	//test, _ := LoadJob(jobID)

	//spew.Dump(test)

	//panic("done")

	return
}

// // CWL: task in WorkflowInstance
// func dbUpdateTaskFields(jobID string, taskID string, updateValue bson.M) (err error) {
// 	session := db.Connection.Session.Copy()
// 	defer session.Close()

// 	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)
// 	selector := bson.M{"id": jobID, "tasks.taskid": taskID}

// 	err = c.Update(selector, bson.M{"$set": updateValue})
// 	if err != nil {
// 		err = fmt.Errorf("(dbUpdateTaskFields) Error updating task: " + err.Error())
// 		return
// 	}
// 	return
// }
