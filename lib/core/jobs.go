package core

import (
	"labix.org/v2/mgo/bson"
)

// Job array type
type Jobs []Job

func (n *Jobs) GetAll(q bson.M) (err error) {
	_, err = dbFind(q, n, nil)
	return
}

func (n *Jobs) GetPaginated(q bson.M, limit int, offset int) (count int, err error) {
	count, err = dbFind(q, n, map[string]int{"limit": limit, "offset": offset})
	return
}

func (n *Jobs) GetAllLimitOffset(q bson.M, limit int, offset int) (err error) {
	_, err = dbFind(q, n, map[string]int{"limit": limit, "offset": offset})
	return
}

func (n *Jobs) GetAllRecent(q bson.M, recent int) (count int, err error) {
	count, err = dbFindSort(q, n, map[string]int{"limit": recent}, "-jid")
	return
}

func (n *Jobs) GetJobAt(index int) Job {
	return (*n)[index]
}

func (n *Jobs) Length() int {
	return len([]Job(*n))
}
