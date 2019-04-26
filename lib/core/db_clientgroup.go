package core

import (
	"errors"

	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/db"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// InitClientGroupDB _
func InitClientGroupDB() {
	session := db.Connection.Session.Copy()
	defer session.Close()
	cc := session.DB(conf.MONGODB_DATABASE).C(conf.DB_COLL_CGS)
	cc.EnsureIndex(mgo.Index{Key: []string{"id"}, Unique: true})
	cc.EnsureIndex(mgo.Index{Key: []string{"name"}, Unique: true})
	cc.EnsureIndex(mgo.Index{Key: []string{"token"}, Unique: true})
}

// dbFindClientGroups _
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

// dbFindSortClientGroups _
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

// LoadClientGroup _
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

// LoadClientGroupByName _
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

// LoadClientGroupByToken _
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

// DeleteClientGroup _
func DeleteClientGroup(id string) (err error) {
	err = dbDelete(bson.M{"id": id}, conf.DB_COLL_CGS)
	return
}
