package core

import (
	//	"errors"
	"errors"
	"github.com/MG-RAST/AWE/lib/db"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

var DB *mgo.Collection

func InitJobDB() {
	DB = db.Connection.DB.C("Jobs")
	DB.EnsureIndex(mgo.Index{Key: []string{"id"}, Unique: true})
}

func dbDelete(q bson.M) (err error) {
	_, err = DB.RemoveAll(q)
	return
}

func dbUpsert(j *Job) (err error) {
	_, err = DB.Upsert(bson.M{"id": j.Id}, &j)
	return
}

func dbFind(q bson.M, results *Jobs, options map[string]int) (count int, err error) {
	query := DB.Find(q)
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
	query := DB.Find(q)
	if count, err = query.Count(); err != nil {
		return 0, err
	}
	if limit, has := options["limit"]; has {
		err = DB.Find(q).Sort(sortby).Limit(limit).All(results)
		return
	}
	err = query.All(results)
	return
}
