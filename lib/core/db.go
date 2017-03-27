package core

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/acl"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/db"
	"github.com/MG-RAST/AWE/lib/logger"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"time"
)

// mongodb has hard limit of 16 MB docuemnt size
var DocumentMaxByte = 16777216

// indexed info fields for search
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
}

func InitClientGroupDB() {
	session := db.Connection.Session.Copy()
	defer session.Close()
	cc := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_CGS)
	cc.EnsureIndex(mgo.Index{Key: []string{"id"}, Unique: true})
	cc.EnsureIndex(mgo.Index{Key: []string{"name"}, Unique: true})
	cc.EnsureIndex(mgo.Index{Key: []string{"token"}, Unique: true})
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
				err = results.Init()
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
		err = results.Init()
		if err != nil {
			return
		}
	}
	return
}

func dbFindSort(q bson.M, results *Jobs, options map[string]int, sortby string) (count int, err error) {
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
				err = results.Init()
				if err != nil {
					return
				}
			}
			return
		}
	}
	err = query.Sort(sortby).All(results)
	if results != nil {
		err = results.Init()
		if err != nil {
			return
		}
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

func dbFindClientGroups(q bson.M, results *ClientGroups) (count int, err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_CGS)
	query := c.Find(q)
	if count, err = query.Count(); err != nil {
		return 0, err
	}
	err = query.All(results)
	return
}

func dbFindSortClientGroups(q bson.M, results *ClientGroups, options map[string]int, sortby string) (count int, err error) {
	if sortby == "" {
		return 0, errors.New("sortby must be an nonempty string")
	}
	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_CGS)
	query := c.Find(q)
	if count, err = query.Count(); err != nil {
		return 0, err
	}

	if limit, has := options["limit"]; has {
		if offset, has := options["offset"]; has {
			err = query.Sort(sortby).Limit(limit).Skip(offset).All(results)
			return
		}
	}
	err = query.Sort(sortby).All(results)
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

	err = dbGetJobTaskField(job_id, task_id, fieldname, result)

	if err != nil {
		return
	}

	//result, ok := myresult.(string)
	//if !ok {
	//	err = fmt.Errorf("(dbGetJobTaskString) Could not read field %s", fieldname)
	//}

	return
}

func dbGetJobTaskInt(job_id string, task_id string, fieldname string) (result int, err error) {

	err = dbGetJobTaskField(job_id, task_id, fieldname, result)

	if err != nil {
		return
	}

	//result, ok := myresult.(int)
	//if !ok {
	//	err = fmt.Errorf("(dbGetJobTaskInt) Could not read field %s", fieldname)
	//}
	return
}

func dbGetJobTaskField(job_id string, task_id string, fieldname string, result interface{}) (err error) {

	// shell example to get task field "remainwork"
	// db.Jobs.find({"id":"e05f1310-ca29-42cc-a765-2dafecf345d4"}, {"tasks" : {"$elemMatch" : {"taskid":"e05f1310-ca29-42cc-a765-2dafecf345d4_0"} }, "tasks.remainwork" : 1}).pretty()

	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)

	selector := bson.M{"id": job_id}

	projection := bson.M{"tasks": bson.M{"$elemMatch": bson.M{"taskid": task_id}}, "tasks." + fieldname: 1}

	temp_result := bson.M{}

	err = c.Find(selector).Select(projection).One(&temp_result)
	if err != nil {
		err = fmt.Errorf("(dbGetJobTaskField) Error getting field from job_id %s , task_id %s and fieldname %s: %s", job_id, task_id, fieldname, err.Error())
		return
	}

	tasks_unknown := temp_result["tasks"]
	tasks, ok := tasks_unknown.([]interface{})

	if !ok {
		err = fmt.Errorf("Array expected, but not found")
		return
	}

	if len(tasks) == 0 {
		err = fmt.Errorf("result task array empty")
		return
	}

	first_task := tasks[0].(bson.M)

	result, ok = first_task[fieldname]

	if !ok {
		err = fmt.Errorf("Field %s not in task object", fieldname)
		return
	}

	logger.Debug(3, "(dbGetJobTaskField) %s got something", fieldname)

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

	logger.Debug(3, "(dbGetPrivateEnv) starting")

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

func DBGetJobAcl(job_id string) (acl acl.Acl, err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)

	selector := bson.M{"id": job_id}

	err = c.Find(selector).One(&acl)
	if err != nil {
		err = fmt.Errorf("Error getting acl field from job_id %s: %s", job_id, err.Error())
		return
	}

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

func dbUpdateJobFields(job_id string, update_value bson.M) (err error) {

	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)

	selector := bson.M{"id": job_id}

	//update_value := bson.M{"tasks.$." + fieldname: value}

	err = c.Update(selector, bson.M{"$set": update_value})
	if err != nil {
		err = fmt.Errorf("Error updating job fields: " + err.Error())
		return
	}

	return
}

func dbUpdateJobFieldString(job_id string, fieldname string, value string) (err error) {

	update_value := bson.M{fieldname: value}

	return dbUpdateJobFields(job_id, update_value)
}

func dbUpdateJobFieldInt(job_id string, fieldname string, value int) (err error) {

	update_value := bson.M{fieldname: value}

	return dbUpdateJobFields(job_id, update_value)
}

func dbIncrementJobField(job_id string, fieldname string, increment_value int) (err error) {

	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)

	selector := bson.M{"id": job_id}

	update_value := bson.M{fieldname: increment_value}

	err = c.Update(selector, bson.M{"$inc": update_value})
	if err != nil {
		err = fmt.Errorf("Error incrementing job_id=%s fieldname=%s by %d: %s", job_id, fieldname, increment_value, err.Error())
		return
	}

	return
}

func dbUpdateJobTaskFields(job_id string, task_id string, update_value bson.M) (err error) {

	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)

	selector := bson.M{"id": job_id, "tasks.taskid": task_id}

	err = c.Update(selector, bson.M{"$set": update_value})
	if err != nil {
		err = fmt.Errorf("(dbUpdateJobTaskFields) Error updating task: " + err.Error())
		return
	}

	return
}

func dbUpdateJobTaskField(job_id string, task_id string, fieldname string, value interface{}) (err error) {

	update_value := bson.M{"tasks.$." + fieldname: value}

	return dbUpdateJobTaskFields(job_id, task_id, update_value)

}

func dbUpdateJobTaskTime(job_id string, task_id string, fieldname string, value time.Time) (err error) {

	update_value := bson.M{"tasks.$." + fieldname: value}

	return dbUpdateJobTaskFields(job_id, task_id, update_value)

}

// deprecated, replaced by more fine-grained modifications
func dbUpdateTask(job_id string, task *Task) (err error) {
	task_id, err := task.GetId()
	if err != nil {
		return
	}

	session := db.Connection.Session.Copy()
	defer session.Close()

	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)

	logger.Debug(3, "(dbUpdateTask) task_id: %s", task_id)

	selector := bson.M{"id": job_id, "tasks.taskid": task_id}

	update_value := bson.M{"tasks.$": task}

	err = c.Update(selector, bson.M{"$set": update_value})
	if err != nil {
		err = fmt.Errorf("(dbUpdateTask) Error updating task: " + err.Error())
		return
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
	if err = c.Find(bson.M{"id": id}).One(&job); err == nil {
		changed, xerr := job.Init()
		if xerr != nil {
			err = xerr
			return
		}

		// To fix incomplete or inconsistent database entries
		if changed {
			job.Save()
		}

		return job, nil
	}
	return nil, err
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

func LoadClientGroup(id string) (clientgroup *ClientGroup, err error) {
	clientgroup = new(ClientGroup)
	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_CGS)
	if err = c.Find(bson.M{"id": id}).One(&clientgroup); err == nil {
		return clientgroup, nil
	}
	return nil, err
}

func LoadClientGroupByName(name string) (clientgroup *ClientGroup, err error) {
	clientgroup = new(ClientGroup)
	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_CGS)
	if err = c.Find(bson.M{"name": name}).One(&clientgroup); err == nil {
		return clientgroup, nil
	}
	return nil, err
}

func LoadClientGroupByToken(token string) (clientgroup *ClientGroup, err error) {
	clientgroup = new(ClientGroup)
	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_CGS)
	if err = c.Find(bson.M{"token": token}).One(&clientgroup); err == nil {
		return clientgroup, nil
	}
	return nil, err
}

func DeleteClientGroup(id string) (err error) {
	err = dbDelete(bson.M{"id": id}, conf.DB_COLL_CGS)
	return
}
