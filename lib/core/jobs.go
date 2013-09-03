package core

import (
	"labix.org/v2/mgo/bson"
)

// Job array type
type Jobs []Job

func (n *Jobs) GetAll(q bson.M) (err error) {
	db, err := DBConnect()
	if err != nil {
		return
	}
	defer db.Close()
	err = db.GetAll(q, n)
	return
}

func (n *Jobs) GetAllLimitOffset(q bson.M, limit int, offset int) (err error) {
	db, err := DBConnect()
	if err != nil {
		return
	}
	defer db.Close()
	err = db.GetAllLimitOffset(q, n, limit, offset)
	return
}

func (n *Jobs) GetAllRecent(q bson.M, recent int) (err error) {
	db, err := DBConnect()
	if err != nil {
		return
	}
	defer db.Close()
	err = db.GetAllRecent(q, n, recent)
	return
}

func (n *Jobs) GetJobAt(index int) Job {
	return (*n)[index]
}

func (n *Jobs) Length() int {
	return len([]Job(*n))
}
