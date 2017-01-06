package core

type WorkunitMap struct {
	RWMutex
	Map map[string]*Workunit
}

func NewWorkunitMap() *WorkunitMap {
	return &WorkunitMap{Map: map[string]*Workunit{}}
}

func (wm *WorkunitMap) Len() int {
	rlock := wm.RLockNamed("WorkunitMap/Len")
	defer wm.RUnlockNamed(rlock)
	return len(wm.Map)
}

func (wm *WorkunitMap) Set(workunit *Workunit) {
	wm.LockNamed("WorkunitMap/Set")
	defer wm.Unlock()
	wm.Map[workunit.Id] = workunit
}

func (wm *WorkunitMap) Get(id string) (workunit *Workunit, ok bool) {
	rlock := wm.RLockNamed("WorkunitMap/Get")
	defer wm.RUnlockNamed(rlock)
	workunit, ok = wm.Map[id]
	return
}

func (wm *WorkunitMap) GetWorkunits() (workunits []*Workunit) {
	workunits = []*Workunit{}
	rlock := wm.RLockNamed("WorkunitMap/GetWorkunits")
	defer wm.RUnlockNamed(rlock)
	for _, workunit := range wm.Map {
		workunits = append(workunits, workunit)
	}

	return
}

func (wm *WorkunitMap) Delete(id string) {
	wm.LockNamed("WorkunitMap/Delete")
	defer wm.Unlock()
	delete(wm.Map, id)
	return
}
