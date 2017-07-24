package core

type WorkunitMap struct {
	RWMutex
	Map map[string]*Workunit
}

func NewWorkunitMap() *WorkunitMap {
	return &WorkunitMap{Map: map[string]*Workunit{}}
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
	wm.Map[workunit.Id] = workunit
	return
}

func (wm *WorkunitMap) Get(id string) (workunit *Workunit, ok bool, err error) {
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

func (wm *WorkunitMap) Delete(id string) (err error) {
	err = wm.LockNamed("WorkunitMap/Delete")
	if err != nil {
		return
	}
	defer wm.Unlock()
	delete(wm.Map, id)
	return
}
