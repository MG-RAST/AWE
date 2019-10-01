package core

import (
	"fmt"

	rwmutex "github.com/MG-RAST/go-rwmutex"
)

//"fmt"

type JobMap struct {
	rwmutex.RWMutex
	_map map[string]*Job
}

func NewJobMap() (t *JobMap) {
	t = &JobMap{
		_map: make(map[string]*Job),
	}
	t.RWMutex.Init("JobMap")
	return t

}

//--------job accessor methods-------

func (jm *JobMap) Len() (length int, err error) {
	read_lock, err := jm.RLockNamed("Len")
	if err != nil {
		return
	}
	defer jm.RUnlockNamed(read_lock)
	length = len(jm._map)
	return
}

func (jm *JobMap) Get(jobid string, lock bool) (job *Job, ok bool, err error) {
	if lock {
		read_lock, xerr := jm.RLockNamed("Get")
		if xerr != nil {
			err = xerr
			return
		}
		defer jm.RUnlockNamed(read_lock)
	}

	job, ok = jm._map[jobid]
	return
}

func (jm *JobMap) Add(job *Job) (err error) {
	err = jm.LockNamed("JobMap/Add")
	if err != nil {
		return
	}
	defer jm.Unlock()

	_, ok := jm._map[job.ID]
	if ok {
		err = fmt.Errorf("(JobMap/Add) %s already in JobMap", job.ID)
		return
	}

	jm._map[job.ID] = job // TODO prevent overwriting
	return
}

func (jm *JobMap) Delete(jobid string, lock bool) (err error) {
	if lock {
		err = jm.LockNamed("Delete")
		if err != nil {
			return
		}
		defer jm.Unlock()
	}
	delete(jm._map, jobid)
	return
}

func (jm *JobMap) Get_List(lock bool) (jobs []*Job, err error) {
	if lock {
		var read_lock rwmutex.ReadLock
		read_lock, err = jm.RLockNamed("Get_List")
		if err != nil {
			return
		}
		defer jm.RUnlockNamed(read_lock)
	}

	jobs = []*Job{}

	for _, job := range jm._map {
		jobs = append(jobs, job)
	}

	return
}
