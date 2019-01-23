package core

import "github.com/MG-RAST/AWE/lib/rwmutex"

//"fmt"
//"github.com/MG-RAST/AWE/lib/logger"

type WorkunitMap struct {
	rwmutex.RWMutex
	Map map[Workunit_Unique_Identifier]*Workunit
}

func NewWorkunitMap() *WorkunitMap {
	return &WorkunitMap{Map: map[Workunit_Unique_Identifier]*Workunit{}}
}

func (wm *WorkunitMap) Len() (length int, err error) {
	rlock, err := wm.RLockNamed("WorkunitMap/Len")
	if err != nil {
		return
	}
	defer wm.RUnlockNamed(rlock)
	length = len(wm.Map)
	return
}

func (wm *WorkunitMap) Set(workunit *Workunit) (err error) {
	err = wm.LockNamed("WorkunitMap/Set")
	if err != nil {
		return
	}
	defer wm.Unlock()

	wm.Map[workunit.Workunit_Unique_Identifier] = workunit
	return
}

func (wm *WorkunitMap) Get(id Workunit_Unique_Identifier) (workunit *Workunit, ok bool, err error) {
	rlock, err := wm.RLockNamed("WorkunitMap/Get")
	if err != nil {
		return
	}
	defer wm.RUnlockNamed(rlock)
	workunit, ok = wm.Map[id]
	return
}

func (wm *WorkunitMap) GetWorkunits() (workunits []*Workunit, err error) {
	workunits = []*Workunit{}
	rlock, err := wm.RLockNamed("WorkunitMap/GetWorkunits")
	if err != nil {
		return
	}
	defer wm.RUnlockNamed(rlock)
	for _, workunit := range wm.Map {
		workunits = append(workunits, workunit)
	}

	return
}

func (wm *WorkunitMap) Delete(id Workunit_Unique_Identifier) (err error) {
	err = wm.LockNamed("WorkunitMap/Delete")
	if err != nil {
		return
	}
	defer wm.Unlock()
	delete(wm.Map, id)
	return
}
