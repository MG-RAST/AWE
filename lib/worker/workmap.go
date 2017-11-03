package worker

import (
	"github.com/MG-RAST/AWE/lib/core"
)

type WorkMap struct {
	core.RWMutex
	_map map[core.Workunit_Unique_Identifier]int
}

func NewWorkMap() *WorkMap {
	wm := &WorkMap{_map: make(map[core.Workunit_Unique_Identifier]int)}
	wm.RWMutex.Init("WorkMap")
	return wm
}

func (this *WorkMap) Get(id core.Workunit_Unique_Identifier) (value int, ok bool, err error) {
	rlock, err := this.RLockNamed("Get")
	if err != nil {
		return
	}
	defer this.RUnlockNamed(rlock)
	value, ok = this._map[id]
	return
}

func (this *WorkMap) GetKeys() (value []core.Workunit_Unique_Identifier, err error) {
	rlock, err := this.RLockNamed("Get")
	if err != nil {
		return
	}
	defer this.RUnlockNamed(rlock)
	for work, _ := range this._map {
		value = append(value, work)
	}
	return
}

func (this *WorkMap) Set(id core.Workunit_Unique_Identifier, value int, name string) (err error) {
	err = this.LockNamed("Set_" + name)
	if err != nil {
		return
	}
	defer this.Unlock()

	this._map[id] = value

	return
}

func (this *WorkMap) Delete(id core.Workunit_Unique_Identifier) (err error) {
	rlock, err := this.RLockNamed("Get")
	if err != nil {
		return
	}
	defer this.RUnlockNamed(rlock)
	delete(this._map, id)
	return
}
