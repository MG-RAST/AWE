package core

import (
	//	"errors"
	"errors"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/db"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

func InitJobDB() {
	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C("Jobs")
	c.EnsureIndex(mgo.Index{Key: []string{"id"}, Unique: true})
}

func dbDelete(q bson.M) (err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C("Jobs")
	_, err = c.RemoveAll(q)
	return
}

func dbUpsert(j *Job) (err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C("Jobs")
	_, err = c.Upsert(bson.M{"id": j.Id}, &j)
	return
}

func dbFind(q bson.M, results *Jobs, options map[string]int) (count int, err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C("Jobs")
	query := c.Find(q)
	if count, err = query.Count(); err != nil {
		return 0, err
	}
	if limit, has := options["limit"]; has {
		if offset, has := options["offset"]; has {
			err = query.Limit(limit).Skip(offset).All(results)
			return
		} else {
			return 0, errors.New("store.db.Find options limit and offset must be used together")
		}
	}
	err = query.All(results)
	return
}

func dbFindSort(q bson.M, results *Jobs, options map[string]int, sortby string) (count int, err error) {
	if sortby == "" {
		return 0, errors.New("sortby must be an nonempty string")
	}
	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C("Jobs")
	query := c.Find(q)
	if count, err = query.Count(); err != nil {
		return 0, err
	}
	if limit, has := options["limit"]; has {
		err = c.Find(q).Sort(sortby).Limit(limit).All(results)
		return
	}
	err = query.All(results)
	return
}

func LoadJob(id string) (job *Job, err error) {
	job = new(Job)
	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C("Jobs")
	if err = c.Find(bson.M{"id": id}).One(&job); err == nil {
		return job, nil
	} else {
		return nil, err
	}
	return nil, err
}
