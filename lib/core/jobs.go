package core

import (
	"gopkg.in/mgo.v2/bson"
)

// Job array type
type Jobs []*Job

func (n *Jobs) Init() {
	for _, job := range *n {
		job.Init()
	}
}

func (n *Jobs) RLockRecursive() {
	for _, job := range *n {
		job.RLockRecursive()
	}
}

func (n *Jobs) RUnlockRecursive() {
	for _, job := range *n {
		job.RUnlockRecursive()
	}
}

func (n *Jobs) GetAllUnsorted(q bson.M) (err error) {
	_, err = dbFind(q, n, nil)
	return
}

func (n *Jobs) GetAll(q bson.M, order string, direction string) (err error) {
	if direction == "desc" {
		order = "-" + order
	}
	_, err = dbFindSort(q, n, nil, order)
	return
}

func (n *Jobs) GetPaginated(q bson.M, limit int, offset int, order string, direction string) (count int, err error) {
	if direction == "desc" {
		order = "-" + order
	}
	count, err = dbFindSort(q, n, map[string]int{"limit": limit, "offset": offset}, order)
	return
}

func (n *Jobs) GetAllLimitOffset(q bson.M, limit int, offset int) (err error) {
	_, err = dbFind(q, n, map[string]int{"limit": limit, "offset": offset})
	return
}

func (n *Jobs) GetAllRecent(q bson.M, recent int) (count int, err error) {
	count, err = dbFindSort(q, n, map[string]int{"limit": recent}, "-updatetime")
	return
}

func (n *Jobs) GetJobAt(index int) *Job {
	return (*n)[index]
}

func (n *Jobs) Length() int {
	return len(*n)
}

func GetJobCount(q bson.M) (count int, err error) {
	count, err = dbCount(q)
	return
}
