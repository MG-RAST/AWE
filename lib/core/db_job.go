package core

import (
	"errors"
	"fmt"
	"time"

	"github.com/MG-RAST/AWE/lib/acl"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/db"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var JobInfoIndexes = []string{"name", "submittime", "completedtime", "pipeline", "clientgroups", "project", "service", "user", "priority", "userattr.submission"}

func HasInfoField(a string) bool {
	for _, b := range JobInfoIndexes {
		if b == a {
			return true
		}
	}
	return false
}

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
	} else {
		return count, nil
	}
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

func dbFindSort(q bson.M, results *Jobs, options map[string]int, sortby string, do_init bool) (count int, err error) {
	if sortby == "" {
		return 0, errors.New("sortby must be an nonempty string")
	}
	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)
	query := c.Find(q)
	if count, err = query.Count(); err != nil {
		return 0, err
	}

	if limit, has := options["limit"]; has {
		if offset, has := options["offset"]; has {
			err = query.Sort(sortby).Limit(limit).Skip(offset).All(results)
			if results != nil {
				if do_init {
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

	var count_changed int

	if do_init {
		count_changed, err = results.Init()
		if err != nil {
			err = fmt.Errorf("(dbFindSort) results.Init() failed: %s", err.Error())
			return
		}
		logger.Debug(1, "%d jobs haven been updated by Init function", count_changed)
	}
	return
}

func DbFindDistinct(q bson.M, d string) (results interface{}, err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)
	err = c.Find(q).Distinct("info."+d, &results)
	return
}

func dbUpdateJobState(job_id string, newState string, notes string) (err error) {
	logger.Debug(3, "(dbUpdateJobState) job_id: %s", job_id)
	var update_value bson.M

	if newState == JOB_STAT_COMPLETED {
		update_value = bson.M{"state": newState, "notes": notes, "info.completedtime": time.Now()}
	} else {
		update_value = bson.M{"state": newState, "notes": notes}
	}
	return dbUpdateJobFields(job_id, update_value)
}

func dbGetJobTasks(job_id string) (tasks []*Task, err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)

	selector := bson.M{"id": job_id}
	fieldname := "tasks"
	err = c.Find(selector).Select(bson.M{fieldname: 1}).All(&tasks)
	if err != nil {
		err = fmt.Errorf("(dbGetJobTasks) Error getting tasks from job_id %s: %s", job_id, err.Error())
		return
	}
	return
}

func dbGetJobTaskString(job_id string, task_id string, fieldname string) (result string, err error) {
	var cont StructContainer
	err = dbGetJobTaskField(job_id, task_id, fieldname, &cont)
	if err != nil {
		return
	}
	result = cont.Data.(string)
	return
}

func dbGetJobTaskTime(job_id string, task_id string, fieldname string) (result time.Time, err error) {
	var cont StructContainer
	err = dbGetJobTaskField(job_id, task_id, fieldname, &cont)
	if err != nil {
		return
	}
	result = cont.Data.(time.Time)
	return
}

func dbGetJobTaskInt(job_id string, task_id string, fieldname string) (result int, err error) {
	var cont StructContainer
	err = dbGetJobTaskField(job_id, task_id, fieldname, &cont)
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
		} else {
			return 0, errors.New("store.db.Find options limit and offset must be used together")
		}
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

func dbUpdateJobFields(job_id string, update_value bson.M) (err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)
	selector := bson.M{"id": job_id}

	err = c.Update(selector, bson.M{"$set": update_value})
	if err != nil {
		err = fmt.Errorf("Error updating job fields: " + err.Error())
		return
	}
	return
}

func dbUpdateJobFieldBoolean(job_id string, fieldname string, value bool) (err error) {
	update_value := bson.M{fieldname: value}
	return dbUpdateJobFields(job_id, update_value)
}

func dbUpdateJobFieldString(job_id string, fieldname string, value string) (err error) {
	update_value := bson.M{fieldname: value}
	return dbUpdateJobFields(job_id, update_value)
}

func dbUpdateJobFieldInt(job_id string, fieldname string, value int) (err error) {
	update_value := bson.M{fieldname: value}
	return dbUpdateJobFields(job_id, update_value)
}

func dbUpdateJobFieldTime(job_id string, fieldname string, value time.Time) (err error) {
	update_value := bson.M{fieldname: value}
	return dbUpdateJobFields(job_id, update_value)
}

func dbUpdateJobFieldNull(job_id string, fieldname string) (err error) {
	update_value := bson.M{fieldname: nil}
	return dbUpdateJobFields(job_id, update_value)
}

func dbUpdateJobTaskFields(job_id string, workflow_instance_id string, task_id string, update_value bson.M) (err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()

	database := conf.DB_COLL_JOBS
	if workflow_instance_id != "" {
		database = conf.DB_COLL_SUBWORKFLOWS
	}

	c := session.DB(conf.MONGODB_DATABASE).C(database)

	var selector bson.M
	var update_op bson.M
	if workflow_instance_id == "" {
		selector = bson.M{"id": job_id, "tasks.taskid": task_id}
		update_op = bson.M{"$set": update_value}
	} else {
		err = fmt.Errorf("not supported")
		return

		unique_id := job_id + workflow_instance_id
		selector = bson.M{"_id": unique_id, "tasks.taskid": task_id}
		update_op = bson.M{"$set": update_value}
	}

	fmt.Println("(dbUpdateJobTaskFields) update_value:")
	spew.Dump(update_value)

	err = c.Update(selector, update_op)
	if err != nil {
		err = fmt.Errorf("(dbUpdateJobTaskFields) (db: %s) Error updating task: %s", database, err.Error())
		return
	}
	return
}

func dbUpdateJobTaskField(job_id string, workflow_instance string, task_id string, fieldname string, value interface{}) (err error) {
	update_value := bson.M{"tasks.$." + fieldname: value}
	err = dbUpdateJobTaskFields(job_id, workflow_instance, task_id, update_value)
	if err != nil {
		err = fmt.Errorf("(dbUpdateJobTaskField) dbUpdateJobTaskFields returned: %s", err.Error())
	}
	return

}

func dbUpdateJobTaskInt(job_id string, workflow_instance string, task_id string, fieldname string, value int) (err error) {
	update_value := bson.M{"tasks.$." + fieldname: value}
	err = dbUpdateJobTaskFields(job_id, workflow_instance, task_id, update_value)
	if err != nil {
		err = fmt.Errorf("(dbUpdateJobTaskInt) dbUpdateJobTaskFields returned: %s", err.Error())
	}
	return

}
func dbUpdateJobTaskBoolean(job_id string, workflow_instance string, task_id string, fieldname string, value bool) (err error) {
	update_value := bson.M{"tasks.$." + fieldname: value}
	err = dbUpdateJobTaskFields(job_id, workflow_instance, task_id, update_value)
	if err != nil {
		err = fmt.Errorf("(dbUpdateJobTaskBoolean) dbUpdateJobTaskFields returned: %s", err.Error())
	}
	return

}

func dbUpdateJobTaskString(job_id string, workflow_instance string, task_id string, fieldname string, value string) (err error) {
	update_value := bson.M{"tasks.$." + fieldname: value}
	err = dbUpdateJobTaskFields(job_id, workflow_instance, task_id, update_value)

	if err != nil {
		err = fmt.Errorf("(dbUpdateJobTaskString) job_id=%s, task_id=%s, fieldname=%s, value=%s dbUpdateJobTaskFields returned: %s", job_id, task_id, fieldname, value, err.Error())
	}
	return
}

func dbUpdateJobTaskTime(job_id string, workflow_instance string, task_id string, fieldname string, value time.Time) (err error) {
	update_value := bson.M{"tasks.$." + fieldname: value}
	err = dbUpdateJobTaskFields(job_id, workflow_instance, task_id, update_value)
	if err != nil {
		err = fmt.Errorf("(dbUpdateJobTaskTime) (fieldname %s) dbUpdateJobTaskFields returned: %s", fieldname, err.Error())
	}
	return
}

func dbUpdateJobTaskPartition(job_id string, workflow_instance string, task_id string, partition *PartInfo) (err error) {
	update_value := bson.M{"tasks.$.partinfo": partition}
	err = dbUpdateJobTaskFields(job_id, workflow_instance, task_id, update_value)
	if err != nil {
		err = fmt.Errorf("(dbUpdateJobTaskPartition) dbUpdateJobTaskFields returned: %s", err.Error())
	}
	return
}

func dbUpdateJobTaskIO(job_id string, workflow_instance string, task_id string, fieldname string, value []*IO) (err error) {
	update_value := bson.M{"tasks.$." + fieldname: value}
	err = dbUpdateJobTaskFields(job_id, workflow_instance, task_id, update_value)
	if err != nil {
		err = fmt.Errorf("(dbUpdateJobTaskIO) dbUpdateJobTaskFields returned: %s", err.Error())
	}
	return
}

func dbIncrementJobTaskField(job_id string, task_id string, fieldname string, increment_value int) (err error) {

	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)

	selector := bson.M{"id": job_id, "tasks.taskid": task_id}

	update_value := bson.M{"tasks.$." + fieldname: increment_value}

	err = c.Update(selector, bson.M{"$inc": update_value})
	if err != nil {
		err = fmt.Errorf("Error incrementing job_id=%s fieldname=%s by %d: %s", job_id, fieldname, increment_value, err.Error())
		return
	}

	return
}

func DbUpdateJobField(job_id string, key string, value interface{}) (err error) {

	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)

	query := bson.M{"id": job_id}
	update_value := bson.M{key: value}
	update := bson.M{"$set": update_value}

	err = c.Update(query, update)
	if err != nil {
		err = fmt.Errorf("Error updating job %s and key %s in job: %s", job_id, key, err.Error())
		return
	}

	return
}

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
	var job_changed bool
	job_changed, err = job.Init() // values have already been set at this point...
	if err != nil {
		err = fmt.Errorf("(LoadJob) job.Init failed: %s", err.Error())
		return
	}

	// B) get WorkflowInstances
	if job.IsCWL {

		c2 := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_SUBWORKFLOWS)

		wis_if := []interface{}{} // have to use interface, because mongo cannot handle interface types

		err = c2.Find(bson.M{"job_id": id}).All(&wis_if)
		if err != nil {
			job = nil
			err = fmt.Errorf("(LoadJob) (DB_COLL_SUBWORKFLOWS) c.Find failed: %s", err.Error())
			return
		}

		if len(wis_if) == 0 {
			err = fmt.Errorf("(LoadJob) no matching WorkflowInstances found for job %s", id)
			return
		}

		var wis []*WorkflowInstance
		wis, err = NewWorkflowInstanceArrayFromInterface(wis_if, job, job.WorkflowContext)
		if err != nil {
			err = fmt.Errorf("(LoadJob) NewWorkflowInstanceArrayFromInterface returned: %s", err.Error())
			return
		}

		if len(wis) == 0 {
			err = fmt.Errorf("(LoadJob) NewWorkflowInstanceArrayFromInterface returned no WorkflowInstances for job %s", id)
			return
		}

		job.WorkflowInstancesMap = make(map[string]*WorkflowInstance)

		for i, _ := range wis {
			var wi_changed bool
			wi := wis[i]
			wi_changed, err = wi.Init(job)
			if err != nil {
				err = fmt.Errorf("(LoadJob) wis[i].Init returned: %s", err.Error())
				return
			}
			if wi_changed {
				err = wi.Save(true)
				if err != nil {
					err = fmt.Errorf("(LoadJob) wi.Save() returned: %s", err.Error())
					return
				}
			}
			// add WorkflowInstance to job
			job.WorkflowInstancesMap[wi.Id] = wi
			// add WorkflowInstance to global*
			err = GlobalWorkflowInstanceMap.Add(wi)
			if err != nil {
				err = fmt.Errorf("(LoadJob) GlobalWorkflowInstanceMap returned: %s", err.Error())
				return
			}
		}
	}

	// To fix incomplete or inconsistent database entries
	if job_changed {
		job.Save()
	}

	return

}

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

func dbGetJobTaskField(job_id string, task_id string, fieldname string, result *StructContainer) (err error) {

	array_name := "tasks"
	id_field := "taskid"
	return dbGetJobArrayField(job_id, task_id, array_name, id_field, fieldname, result)
}

// TODO: warning: this does not cope with subfields such as "partinfo.index"
func dbGetJobArrayField(job_id string, task_id string, array_name string, id_field string, fieldname string, result *StructContainer) (err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)
	selector := bson.M{"id": job_id}

	projection := bson.M{array_name: bson.M{"$elemMatch": bson.M{id_field: task_id}}, array_name + "." + fieldname: 1}
	temp_result := bson.M{}

	err = c.Find(selector).Select(projection).One(&temp_result)
	if err != nil {
		err = fmt.Errorf("(dbGetJobArrayField) Error getting field from job_id %s , %s=%s and fieldname %s: %s", job_id, id_field, task_id, fieldname, err.Error())
		return
	}

	//logger.Debug(3, "GOT: %v", temp_result)

	tasks_unknown := temp_result[array_name]
	tasks, ok := tasks_unknown.([]interface{})

	if !ok {
		err = fmt.Errorf("(dbGetJobArrayField) Array expected, but not found")
		return
	}
	if len(tasks) == 0 {
		err = fmt.Errorf("(dbGetJobArrayField) result task array empty")
		return
	}

	first_task := tasks[0].(bson.M)
	test_result, ok := first_task[fieldname]

	if !ok {
		err = fmt.Errorf("(dbGetJobArrayField) Field %s not in task object", fieldname)
		return
	}
	result.Data = test_result

	//logger.Debug(3, "GOT2: %v", result)
	logger.Debug(3, "(dbGetJobArrayField) %s got something", fieldname)
	return
}

func dbGetJobTask(job_id string, task_id string) (result *Task, err error) {
	dummy_job := NewJob()
	dummy_job.Init()

	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)

	selector := bson.M{"id": job_id}
	projection := bson.M{"tasks": bson.M{"$elemMatch": bson.M{"taskid": task_id}}}

	err = c.Find(selector).Select(projection).One(&dummy_job)
	if err != nil {
		err = fmt.Errorf("(dbGetJobTask) Error getting field from job_id %s , task_id %s : %s", job_id, task_id, err.Error())
		return
	}
	if len(dummy_job.Tasks) != 1 {
		err = fmt.Errorf("(dbGetJobTask) len(dummy_job.Tasks) != 1   len(dummy_job.Tasks)=%d", len(dummy_job.Tasks))
		return
	}

	result = dummy_job.Tasks[0]
	return
}

func dbGetPrivateEnv(job_id string, task_id string) (result map[string]string, err error) {
	task, err := dbGetJobTask(job_id, task_id)
	if err != nil {
		return
	}
	result = task.Cmd.Environ.Private
	for key, val := range result {
		logger.Debug(3, "(dbGetPrivateEnv) got %s=%s ", key, val)
	}
	return
}

func dbGetJobFieldInt(job_id string, fieldname string) (result int, err error) {
	err = dbGetJobField(job_id, fieldname, &result)
	if err != nil {
		return
	}
	return
}

func dbGetJobFieldString(job_id string, fieldname string) (result string, err error) {
	err = dbGetJobField(job_id, fieldname, &result)
	logger.Debug(3, "(dbGetJobFieldString) result: %s", result)
	if err != nil {
		return
	}
	return
}

func dbGetJobField(job_id string, fieldname string, result interface{}) (err error) {

	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)
	selector := bson.M{"id": job_id}

	err = c.Find(selector).Select(bson.M{fieldname: 1}).One(&result)
	if err != nil {
		err = fmt.Errorf("(dbGetJobField) Error getting field from job_id %s and fieldname %s: %s", job_id, fieldname, err.Error())
		return
	}
	return
}

type Job_Acl struct {
	Acl acl.Acl `bson:"acl" json:"-"`
}

func DBGetJobAcl(job_id string) (_acl acl.Acl, err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)
	selector := bson.M{"id": job_id}

	job := Job_Acl{}

	err = c.Find(selector).Select(bson.M{"acl": 1}).One(&job)
	if err != nil {
		err = fmt.Errorf("Error getting acl field from job_id %s: %s", job_id, err.Error())
		return
	}

	_acl = job.Acl
	return
}

func dbGetJobFieldTime(job_id string, fieldname string) (result time.Time, err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)
	selector := bson.M{"id": job_id}

	err = c.Find(selector).Select(bson.M{fieldname: 1}).One(&result)
	if err != nil {
		err = fmt.Errorf("(dbGetJobFieldTime) Error getting field from job_id %s and fieldname %s: %s", job_id, fieldname, err.Error())
		return
	}
	return
}

func dbPushJobTask(job_id string, task *Task) (err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)

	selector := bson.M{"id": job_id}
	change := bson.M{"$push": bson.M{"tasks": task}}

	err = c.Update(selector, change)
	if err != nil {
		err = fmt.Errorf("Error adding task: " + err.Error())
		return
	}

	//fmt.Printf("(dbPushJobTask) added task: %s\n", task.Task_Unique_Identifier.TaskName)
	//test, _ := LoadJob(job_id)

	//spew.Dump(test)

	//panic("done")

	return
}

// // CWL: task in WorkflowInstance
// func dbUpdateTaskFields(job_id string, task_id string, update_value bson.M) (err error) {
// 	session := db.Connection.Session.Copy()
// 	defer session.Close()

// 	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)
// 	selector := bson.M{"id": job_id, "tasks.taskid": task_id}

// 	err = c.Update(selector, bson.M{"$set": update_value})
// 	if err != nil {
// 		err = fmt.Errorf("(dbUpdateTaskFields) Error updating task: " + err.Error())
// 		return
// 	}
// 	return
// }
