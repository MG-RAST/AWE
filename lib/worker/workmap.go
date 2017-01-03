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

func (this *WorkMap) Get(id string) (value int, ok bool) {
	rlock := this.RLockNamed("Get")
	defer this.RUnlockNamed(rlock)
	value, ok = this._map[id]
	return
}

func (this *WorkMap) Set(id string, value int, name string) {
	this.LockNamed("Set_" + name)
	defer this.Unlock()

	this._map[id] = value

	return
}

func (this *WorkMap) Delete(id string) {
	rlock := this.RLockNamed("Get")
	defer this.RUnlockNamed(rlock)
	delete(this._map, id)
	return
}
