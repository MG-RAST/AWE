package core

import (
	"errors"

	"github.com/MG-RAST/AWE/lib/logger"
	"sort"
	"sync"
)

type WorkQueue struct {
	sync.RWMutex
	//workMap  map[string]*Workunit //all parsed workunits
	workMap  WorkunitMap
	Wait     WorkunitMap //ids of waiting workunits
	Checkout WorkunitMap //ids of workunits being checked out
	Suspend  WorkunitMap //ids of suspended workunits
}

func NewWorkQueue() *WorkQueue {
	wq := &WorkQueue{
		workMap:  *NewWorkunitMap(),
		Wait:     *NewWorkunitMap(),
		Checkout: *NewWorkunitMap(),
		Suspend:  *NewWorkunitMap(),
	}

	wq.workMap.Init("WorkQueue/workMap")
	wq.Wait.Init("WorkQueue/Wait")
	wq.Checkout.Init("WorkQueue/Checkout")
	wq.Suspend.Init("WorkQueue/Suspend")

	return wq
}

func (wq *WorkQueue) Len() int {
	return wq.workMap.Len()
}

//func (wq *WorkQueue) WaitLen() (l int) {
//	wq.RLock()
//	l = len(wq.wait)
//	wq.RUnlock()
//	return
//}

//--------accessor methods-------

//func (wq *WorkQueue) copyWork(a *Workunit) (b *Workunit) {
//	b = new(Workunit)
//	*b = *a
//	return
//}

func (wq *WorkQueue) Add(workunit *Workunit) (err error) {
	if workunit.Id == "" {
		return errors.New("try to push a workunit with an empty id")
	}
	wq.Lock()
	defer wq.Unlock()
	id := workunit.Id
	//wq.workMap[id] = workunit
	wq.workMap.Set(workunit)
	wq.Wait.Set(workunit)
	wq.Checkout.Delete(id)
	wq.Suspend.Delete(id)
	//wq.workMap[id].State = WORK_STAT_QUEUED
	workunit.State = WORK_STAT_QUEUED
	return nil
}

func (wq *WorkQueue) Put(workunit *Workunit) {
	//wq.Lock()
	//wq.workMap[workunit.Id] = workunit
	//wq.Unlock()
	wq.workMap.Set(workunit)
}

func (wq *WorkQueue) Get(id string) (w *Workunit, ok bool) {

	w, ok = wq.workMap.Get(id)

	return

	//if work, ok := wq.workMap.Get(id); ok {
	//	copy := wq.copyWork(work)
	//	return copy, true
	//}
	//return nil, false
}

// TODO remove this function!
//func (wq *WorkQueue) GetSet(workids []string) (worklist []*Workunit) {

//	for _, id := range workids {
//		if work, ok := wq.workMap.Get(id); ok {
//			copy := wq.copyWork(work)
//			worklist = append(worklist, copy)
//		}
//	}
//	return
//}

func (wq *WorkQueue) GetForJob(jobid string) (worklist []*Workunit) {

	for _, work := range wq.workMap.GetWorkunits() {

		if jobid == getParentJobId(work.Id) {
			worklist = append(worklist, work)
		}
	}
	return
}

func (wq *WorkQueue) GetAll() (worklist []*Workunit) {

	return wq.workMap.GetWorkunits()
}

func (wq *WorkQueue) Clean() (workids []string) {
	wq.Lock()
	defer wq.Unlock()
	for _, work := range wq.workMap.GetWorkunits() {
		id := work.Id
		if work == nil || work.Info == nil {
			workids = append(workids, id)
			wq.Wait.Delete(id)
			wq.Checkout.Delete(id)
			wq.Suspend.Delete(id)
			wq.workMap.Delete(id)
			logger.Error("error: in WorkQueue workunit %s is nil, deleted from queue", id)
		}
	}
	return
}

func (wq *WorkQueue) Delete(id string) {
	wq.Lock()
	wq.Wait.Delete(id)
	wq.Checkout.Delete(id)
	wq.Suspend.Delete(id)
	wq.workMap.Delete(id)
	wq.Unlock()
}

func (wq *WorkQueue) Has(id string) (has bool) {

	_, has = wq.workMap.Get(id)

	//wq.RLock()
	//defer wq.RUnlock()
	//if work, ok := wq.workMap[id]; ok {
	//	if work == nil {
	//		logger.Error("error: in WorkQueue workunit %s is nil", id)
	//		has = false
	//	} else {
	//		has = true
	//	}
	//} else {
	//	has = false
	//}
	return
}

// TODO remove this ???
//func (wq *WorkQueue) List() (workids []string) {
//	wq.RLock()
//	defer wq.RUnlock()
//	for id, _ := range wq.workMap.Map {
//		workids = append(workids, id)
//	}
//	return
//}

//func (wq *WorkQueue) WaitList() (workids []string) {
//	wq.RLock()
//	defer wq.RUnlock()
//	for id, _ := range wq.wait {
//		workids = append(workids, id)
//	}
//	return
//}

//--------end of accessors-------

func (wq *WorkQueue) StatusChange(id string, new_status string) (err error) {
	//move workunit id between maps. no need to care about the old status because
	//delete function will do nothing if the operated map has no such key.
	wq.Lock()
	defer wq.Unlock()
	workunit, ok := wq.workMap.Get(id)
	if !ok {
		return errors.New("WQueue.statusChange: invalid workunit id:" + id)
	}
	if new_status == WORK_STAT_CHECKOUT {
		wq.Checkout.Set(workunit)
		wq.Wait.Delete(id)
		wq.Suspend.Delete(id)
	} else if new_status == WORK_STAT_QUEUED {
		wq.Wait.Set(workunit)
		wq.Checkout.Delete(id)
		wq.Suspend.Delete(id)
	} else if new_status == WORK_STAT_SUSPEND {
		wq.Suspend.Set(workunit)
		wq.Checkout.Delete(id)
		wq.Wait.Delete(id)
	} else {
		return errors.New("WorkQueue.statusChange: invalid new status:" + new_status)
	}
	workunit.State = new_status
	return
}

//select workunits, return a slice of ids based on given queuing policy and requested count
//if available is a positive value, filter by workunit input size
func (wq *WorkQueue) selectWorkunits(workunits WorkList, policy string, available int64, count int) (selected []*Workunit, err error) {
	logger.Debug(3, "starting selectWorkunits")

	if policy == "FCFS" {
		sort.Sort(byFCFS{workunits})
	}
	added := 0
	for _, work := range workunits {
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
	logger.Debug(3, "done with selectWorkunits")
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
