package core

import (
	"fmt"

	"github.com/MG-RAST/AWE/lib/logger"
	"gopkg.in/mgo.v2/bson"
)

// Job array type
type Jobs []*Job

func (n *Jobs) Init() (changed_count int, err error) {
	changed_count = 0
	for _, job := range *n {
		changed, xerr := job.Init("")
		if xerr != nil {
			err = fmt.Errorf("(jobs.Init) job.Init() returned %s", xerr.Error())
			logger.Error(err.Error())
			continue
			//return
		}
		if changed {
			changed_count += 1
			err = job.Save()
			if err != nil {
				err = fmt.Errorf("(jobs.Init) job.Save() returns: %s", err.Error())
				return
			}
		}
	}
	return
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

func (n *Jobs) GetAll(q bson.M, order string, direction string, do_init bool) (err error) {
	if direction == "desc" {
		order = "-" + order
	}
	_, err = dbFindSort(q, n, nil, order, do_init)
	return
}

func (n *Jobs) GetPaginated(q bson.M, limit int, offset int, order string, direction string, do_init bool) (count int, err error) {
	if direction == "desc" {
		order = "-" + order
	}
	count, err = dbFindSort(q, n, map[string]int{"limit": limit, "offset": offset}, order, do_init)
	return
}

func (n *Jobs) GetAllLimitOffset(q bson.M, limit int, offset int) (err error) {
	_, err = dbFind(q, n, map[string]int{"limit": limit, "offset": offset})
	return
}

func (n *Jobs) GetAllRecent(q bson.M, recent int, do_init bool) (count int, err error) {
	count, err = dbFindSort(q, n, map[string]int{"limit": recent}, "-updatetime", do_init)
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

// patch the admin view data function from the job controller through to the db.go
func GetAdminView(special string) (data []interface{}, err error) {
	data, err = dbAdminData(special)
	return
}
