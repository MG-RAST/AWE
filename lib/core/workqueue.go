package core

import (
	"errors"

	"github.com/MG-RAST/AWE/lib/logger"
	"sort"
	//"sync"
)

type WorkQueue struct {
	//sync.RWMutex
	//workMap  map[string]*Workunit //all parsed workunits
	all      WorkunitMap
	Queue    WorkunitMap // WORK_STAT_QUEUED - waiting workunits
	Checkout WorkunitMap // WORK_STAT_CHECKOUT - workunits being checked out
	Suspend  WorkunitMap // WORK_STAT_SUSPEND - suspended workunits
}

func NewWorkQueue() *WorkQueue {
	wq := &WorkQueue{
		all:      *NewWorkunitMap(),
		Queue:    *NewWorkunitMap(),
		Checkout: *NewWorkunitMap(),
		Suspend:  *NewWorkunitMap(),
	}

	wq.all.Init("WorkQueue/workMap")
	wq.Queue.Init("WorkQueue/Queue")
	wq.Checkout.Init("WorkQueue/Checkout")
	wq.Suspend.Init("WorkQueue/Suspend")

	return wq
}

//--------accessor methods-------

func (wq *WorkQueue) Len() (int, error) {
	return wq.all.Len()
}

func (wq *WorkQueue) Add(workunit *Workunit) (err error) {
	if workunit.Id == "" {
		return errors.New("try to push a workunit with an empty id")
	}

	//id := workunit.Id

	err = wq.all.Set(workunit)
	if err != nil {
		return
	}

	err = wq.StatusChange("", workunit, WORK_STAT_QUEUED)
	if err != nil {
		return
	}

	return
}

func (wq *WorkQueue) Get(id string) (w *Workunit, ok bool, err error) {

	w, ok, err = wq.all.Get(id)

	return

}

func (wq *WorkQueue) GetForJob(jobid string) (worklist []*Workunit, err error) {

	workunits, err := wq.all.GetWorkunits()
	if err != nil {
		return
	}
	for _, work := range workunits {

		if jobid == getParentJobId(work.Id) {
			worklist = append(worklist, work)
		}
	}
	return
}

func (wq *WorkQueue) GetAll() (worklist []*Workunit, err error) {

	return wq.all.GetWorkunits()
}

func (wq *WorkQueue) Clean() (workids []string) {
	//wq.Lock()
	//defer wq.Unlock()
	workunt_list, err := wq.all.GetWorkunits()
	if err != nil {
		return
	}
	for _, work := range workunt_list {
		id := work.Id
		if work == nil || work.Info == nil {
			workids = append(workids, id)
			wq.Queue.Delete(id)
			wq.Checkout.Delete(id)
			wq.Suspend.Delete(id)
			wq.all.Delete(id)
			logger.Error("error: in WorkQueue workunit %s is nil, deleted from queue", id)
		}
	}
	return
}

func (wq *WorkQueue) Delete(id string) (err error) {

	_ = wq.Queue.Delete(id)

	_ = wq.Checkout.Delete(id)

	_ = wq.Suspend.Delete(id)

	_ = wq.all.Delete(id)

	return

}

func (wq *WorkQueue) Has(id string) (has bool, err error) {

	_, has, err = wq.all.Get(id)

	return
}

//--------end of accessors-------

func (wq *WorkQueue) StatusChange(id string, workunit *Workunit, new_status string) (err error) {
	//move workunit id between maps. no need to care about the old status because
	//delete function will do nothing if the operated map has no such key.

	if workunit == nil {
		var ok bool
		workunit, ok, err = wq.all.Get(id)
		if err != nil {
			return
		}
		if !ok {
			return errors.New("WQueue.statusChange: invalid workunit id:" + id)
		}
	}

	if workunit.State == new_status {
		return
	}

	switch new_status {
	case WORK_STAT_CHECKOUT:
		wq.Queue.Delete(id)
		wq.Suspend.Delete(id)
		workunit.SetState(new_status)
		wq.Checkout.Set(workunit)
	case WORK_STAT_QUEUED:
		wq.Checkout.Delete(id)
		wq.Suspend.Delete(id)
		workunit.SetState(new_status)
		wq.Queue.Set(workunit)

	case WORK_STAT_SUSPEND:
		wq.Checkout.Delete(id)
		wq.Queue.Delete(id)
		workunit.SetState(new_status)
		wq.Suspend.Set(workunit)

	default:
		return errors.New("WorkQueue.statusChange: invalid new status:" + new_status)
	}

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
