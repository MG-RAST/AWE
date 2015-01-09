package core

import (
	"github.com/MG-RAST/golib/mgo/bson"
)

// Job array type
type Jobs []Job

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

func (n *Jobs) GetJobAt(index int) Job {
	return (*n)[index]
}

func (n *Jobs) Length() int {
	return len([]Job(*n))
}

func GetJobCount(q bson.M) (count int, err error) {
	count, err = dbCount(q)
	return
}
