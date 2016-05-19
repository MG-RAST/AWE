package core

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/logger"
	"sort"
	"sync"
)

type wQueueShow struct {
	WorkMap  map[string]*Workunit `bson:"workmap" json:"workmap"`
	Wait     map[string]bool      `bson:"wait" json:"wait"`
	Checkout map[string]bool      `bson:"checkout" json:"checkout"`
	Suspend  map[string]bool      `bson:"suspend" json:"suspend"`
}

type WQueue struct {
	sync.RWMutex
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

func (wq *WQueue) Len() (l int) {
	wq.RLock()
	l = len(wq.workMap)
	wq.RUnlock()
	return
}

func (wq *WQueue) WaitLen() (l int) {
	wq.RLock()
	l = len(wq.wait)
	wq.RUnlock()
	return
}

func (wq *WQueue) CheckoutLen() (l int) {
	wq.RLock()
	l = len(wq.checkout)
	wq.RUnlock()
	return
}

func (wq *WQueue) SuspendLen() (l int) {
	wq.RLock()
	l = len(wq.suspend)
	wq.RUnlock()
	return
}

//--------accessor methods-------

func (wq *WQueue) copyWork(a *Workunit) (b *Workunit) {
	b = new(Workunit)
	*b = *a
	return
}

func (wq *WQueue) Add(workunit *Workunit) (err error) {
	if workunit.Id == "" {
		return errors.New("try to push a workunit with an empty id")
	}
	wq.Lock()
	defer wq.Unlock()
	id := workunit.Id
	wq.workMap[id] = workunit
	wq.wait[id] = true
	delete(wq.checkout, id)
	delete(wq.suspend, id)
	wq.workMap[id].State = WORK_STAT_QUEUED
	return nil
}

func (wq *WQueue) Put(workunit *Workunit) {
	wq.Lock()
	wq.workMap[workunit.Id] = workunit
	wq.Unlock()
}

func (wq *WQueue) Get(id string) (*Workunit, bool) {
	wq.RLock()
	defer wq.RUnlock()
	if work, ok := wq.workMap[id]; ok {
		copy := wq.copyWork(work)
		return copy, true
	}
	return nil, false
}

func (wq *WQueue) GetSet(workids []string) (worklist []*Workunit) {
	wq.RLock()
	defer wq.RUnlock()
	for _, id := range workids {
		if work, ok := wq.workMap[id]; ok {
			copy := wq.copyWork(work)
			worklist = append(worklist, copy)
		}
	}
	return
}

func (wq *WQueue) GetForJob(jobid string) (worklist []*Workunit) {
	wq.RLock()
	defer wq.RUnlock()
	for id, work := range wq.workMap {
		if jobid == getParentJobId(id) {
			copy := wq.copyWork(work)
			worklist = append(worklist, copy)
		}
	}
	return
}

func (wq *WQueue) GetAll() (worklist []*Workunit) {
	wq.RLock()
	defer wq.RUnlock()
	for _, work := range wq.workMap {
		copy := wq.copyWork(work)
		worklist = append(worklist, copy)
	}
	return
}

func (wq *WQueue) Clean() (workids []string) {
	wq.Lock()
	defer wq.Unlock()
	for id, work := range wq.workMap {
		if work == nil || work.Info == nil {
			workids = append(workids, id)
			delete(wq.wait, id)
			delete(wq.checkout, id)
			delete(wq.suspend, id)
			delete(wq.workMap, id)
			logger.Error(fmt.Sprintf("error: in WQueue workunit %s is nil, deleted from queue", id))
		}
	}
	return
}

func (wq *WQueue) Delete(id string) {
	wq.Lock()
	delete(wq.wait, id)
	delete(wq.checkout, id)
	delete(wq.suspend, id)
	delete(wq.workMap, id)
	wq.Unlock()
}

func (wq *WQueue) Has(id string) (has bool) {
	wq.RLock()
	defer wq.RUnlock()
	if work, ok := wq.workMap[id]; ok {
		if work == nil {
			logger.Error(fmt.Sprintf("error: in WQueue workunit %s is nil", id))
			has = false
		} else {
			has = true
		}
	} else {
		has = false
	}
	return
}

func (wq *WQueue) List() (workids []string) {
	wq.RLock()
	defer wq.RUnlock()
	for id, _ := range wq.workMap {
		workids = append(workids, id)
	}
	return
}

func (wq *WQueue) WaitList() (workids []string) {
	wq.RLock()
	defer wq.RUnlock()
	for id, _ := range wq.wait {
		workids = append(workids, id)
	}
	return
}

//--------end of accessors-------

func (wq *WQueue) StatusChange(id string, new_status string) (err error) {
	if _, ok := wq.workMap[id]; !ok {
		return errors.New("WQueue.statusChange: invalid workunit id:" + id)
	}
	//move workunit id between maps. no need to care about the old status because
	//delete function will do nothing if the operated map has no such key.
	wq.Lock()
	defer wq.Unlock()
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
//if available is a positive value, filter by workunit input size
func (wq *WQueue) selectWorkunits(workid []string, policy string, available int64, count int) (selected []*Workunit, err error) {
	logger.Debug(3, fmt.Sprintf("starting selectWorkunits"))
	worklist := wq.GetSet(workid)
	if policy == "FCFS" {
		sort.Sort(byFCFS{worklist})
	}
	added := 0
	for _, work := range worklist {
		if added == count {
			break
		}
		inputSize := int64(0)
		for _, input := range work.Inputs {
			inputSize = inputSize + input.Size
		}
		// skip work that is too large for client
		if (available < 0) || (available > inputSize) {
			selected = append(selected, work)
			added = added + 1
		}
	}
	logger.Debug(3, fmt.Sprintf("done with selectWorkunits"))
	return
}

//queuing/prioritizing related functions
type WorkList []*Workunit

func (wl WorkList) Len() int      { return len(wl) }
func (wl WorkList) Swap(i, j int) { wl[i], wl[j] = wl[j], wl[i] }

type byFCFS struct{ WorkList }

//compare priority first, then FCFS (if priorities are the same)
func (s byFCFS) Less(i, j int) (ret bool) {
	p_i := s.WorkList[i].Info.Priority
	p_j := s.WorkList[j].Info.Priority
	switch {
	case p_i > p_j:
		return true
	case p_i < p_j:
		return false
	case p_i == p_j:
		return s.WorkList[i].Info.SubmitTime.Before(s.WorkList[j].Info.SubmitTime)
	}
	return
}
