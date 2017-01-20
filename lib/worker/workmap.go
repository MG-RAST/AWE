package worker

import (
	"github.com/MG-RAST/AWE/lib/core"
)

type WorkMap struct {
	core.RWMutex
	_map map[string]int
}

func NewWorkMap() *WorkMap {
	wm := &WorkMap{_map: make(map[string]int)}
	wm.RWMutex.Init("WorkMap")
	return wm
}

func (this *WorkMap) Get(id string) (value int, ok bool, err error) {
	rlock, err := this.RLockNamed("Get")
	if err != nil {
		return
	}
	defer this.RUnlockNamed(rlock)
	value, ok = this._map[id]
	return
}

func (this *WorkMap) Set(id string, value int, name string) (err error) {
	err = this.LockNamed("Set_" + name)
	if err != nil {
		return
	}
	defer this.Unlock()

	this._map[id] = value

	return
}

func (this *WorkMap) Delete(id string) (err error) {
	rlock, err := this.RLockNamed("Get")
	if err != nil {
		return
	}
	defer this.RUnlockNamed(rlock)
	delete(this._map, id)
	return
}
