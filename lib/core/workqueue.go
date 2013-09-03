package core

import (
	"errors"
	"sort"
)

type WQueue struct {
	workMap  map[string]*Workunit //all parsed workunits
	wait     map[string]bool      //ids of waiting workunits
	checkout map[string]bool      //ids of workunits being checked out
	suspend  map[string]bool      //ids of suspended workunits
}

func NewWQueue() *WQueue {
	return &WQueue{
		workMap:  map[string]*Workunit{},
		wait:     map[string]bool{},
		checkout: map[string]bool{},
		suspend:  map[string]bool{},
	}
}

func (wq WQueue) Len() int {
	return len(wq.workMap)
}

func (wq *WQueue) Add(workunit *Workunit) (err error) {
	if workunit.Id == "" {
		return errors.New("try to push a workunit with an empty id")
	}
	wq.workMap[workunit.Id] = workunit
	wq.StatusChange(workunit.Id, WORK_STAT_QUEUED)
	return nil
}

func (wq *WQueue) Get(id string) (work *Workunit, ok bool) {
	work, ok = wq.workMap[id]
	return
}

func (wq *WQueue) Delete(id string) {
	delete(wq.wait, id)
	delete(wq.checkout, id)
	delete(wq.suspend, id)
	delete(wq.workMap, id)
}

func (wq *WQueue) Has(id string) (has bool) {
	if _, ok := wq.workMap[id]; ok {
		has = true
	} else {
		has = false
	}
	return
}

func (wq *WQueue) StatusChange(id string, new_status string) (err error) {
	if _, ok := wq.workMap[id]; !ok {
		return errors.New("WQueue.statusChange: invalid workunit id:" + id)
	}
	//move workunit id between maps. no need to care about the old status because
	//delete function will do nothing if the operated map has no such key.
	if new_status == WORK_STAT_CHECKOUT {
		wq.checkout[id] = true
		delete(wq.wait, id)
		delete(wq.suspend, id)
	} else if new_status == WORK_STAT_QUEUED {
		wq.wait[id] = true
		delete(wq.checkout, id)
		delete(wq.suspend, id)
	} else if new_status == WORK_STAT_SUSPEND {
		wq.suspend[id] = true
		delete(wq.checkout, id)
		delete(wq.wait, id)
	} else {
		return errors.New("WQueue.statusChange: invalid new status:" + new_status)
	}
	wq.workMap[id].State = new_status
	return
}

//select workunits, return a slice of ids based on given queuing policy and requested count
func (wq *WQueue) selectWorkunits(workid []string, policy string, count int) (selected []*Workunit, err error) {
	worklist := []*Workunit{}
	for _, id := range workid {
		worklist = append(worklist, wq.workMap[id])
	}
	//selected = []*Workunit{}
	if policy == "FCFS" {
		sort.Sort(byFCFS{worklist})
	}
	for i := 0; i < count; i++ {
		selected = append(selected, worklist[i])
	}
	return
}

//queuing/prioritizing related functions
type WorkList []*Workunit

func (wl WorkList) Len() int      { return len(wl) }
func (wl WorkList) Swap(i, j int) { wl[i], wl[j] = wl[j], wl[i] }

type byFCFS struct{ WorkList }

func (s byFCFS) Less(i, j int) bool {
	return s.WorkList[i].Info.SubmitTime.Before(s.WorkList[j].Info.SubmitTime)
}
