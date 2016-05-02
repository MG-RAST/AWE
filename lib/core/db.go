package core

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/db"
	mgo "github.com/MG-RAST/AWE/vendor/gopkg.in/mgo.v2"
	"github.com/MG-RAST/AWE/vendor/gopkg.in/mgo.v2/bson"
)

// mongodb has hard limit of 16 MB docuemnt size
var DocumentMaxByte = 16777216

func InitJobDB() {
	session := db.Connection.Session.Copy()
	defer session.Close()
	cj := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)
	cj.EnsureIndex(mgo.Index{Key: []string{"acl.owner"}, Background: true})
	cj.EnsureIndex(mgo.Index{Key: []string{"acl.read"}, Background: true})
	cj.EnsureIndex(mgo.Index{Key: []string{"acl.write"}, Background: true})
	cj.EnsureIndex(mgo.Index{Key: []string{"acl.delete"}, Background: true})
	cj.EnsureIndex(mgo.Index{Key: []string{"id"}, Unique: true})
	cj.EnsureIndex(mgo.Index{Key: []string{"info.submittime"}, Background: true})
	cj.EnsureIndex(mgo.Index{Key: []string{"info.completedtime"}, Background: true})
	cj.EnsureIndex(mgo.Index{Key: []string{"info.pipeline"}, Background: true})
	cj.EnsureIndex(mgo.Index{Key: []string{"info.user"}, Background: true})
	cj.EnsureIndex(mgo.Index{Key: []string{"jid"}, Background: true})
	cj.EnsureIndex(mgo.Index{Key: []string{"state"}, Background: true})
	cj.EnsureIndex(mgo.Index{Key: []string{"updatetime"}, Background: true})
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
	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)
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

func LoadJob(id string) (job *Job, err error) {
	job = new(Job)
	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_JOBS)
	if err = c.Find(bson.M{"id": id}).One(&job); err == nil {
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

func DeleteJob(id string) (err error) {
	err = dbDelete(bson.M{"id": id}, conf.DB_COLL_JOBS)
	return
}

func DeleteClientGroup(id string) (err error) {
	err = dbDelete(bson.M{"id": id}, conf.DB_COLL_CGS)
	return
}
