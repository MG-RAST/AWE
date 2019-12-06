package core

import (
	"fmt"

	rwmutex "github.com/MG-RAST/go-rwmutex"
)

type WorkunitList struct {
	rwmutex.RWMutex `bson:"-" json:"-"`
	_map            map[Workunit_Unique_Identifier]bool
	Data            []string `json:"data"`
}

func NewWorkunitList() *WorkunitList {
	return &WorkunitList{_map: make(map[Workunit_Unique_Identifier]bool)}

}

func (this *WorkunitList) Init(name string) {
	this.RWMutex.Init(name)
	if this._map == nil {
		this._map = make(map[Workunit_Unique_Identifier]bool)
	}
	if this.Data == nil {
		this.Data = []string{}
	}
}

// lock always
func (cl *WorkunitList) Add(workid Workunit_Unique_Identifier) (err error) {

	err = cl.LockNamed("Add")
	if err != nil {
		return
	}
	defer cl.Unlock()

	cl._map[workid] = true
	cl.sync()

	return
}

func (cl *WorkunitList) Length(lock bool) (clength int, err error) {
	if lock {
		read_lock, xerr := cl.RLockNamed("Length")
		if xerr != nil {
			err = xerr
			return
		}
		defer cl.RUnlockNamed(read_lock)
	}
	clength = len(cl.Data)

	return
}

// IsEmpty _
func (cl *WorkunitList) IsEmpty(lock bool) (empty bool, err error) {
	if lock {
		readLock, xerr := cl.RLockNamed("IsEmpty")
		if xerr != nil {
			err = xerr
			return
		}
		defer cl.RUnlockNamed(readLock)
	}
	if len(cl.Data) == 0 {
		empty = true
		return
	}
	empty = false

	return
}

func (cl *WorkunitList) Delete(workid Workunit_Unique_Identifier, writeLock bool) (err error) {
	if writeLock {
		err = cl.LockNamed("Delete")
		defer cl.Unlock()
	}
	delete(cl._map, workid)
	cl.sync()
	return
}

func (cl *WorkunitList) Delete_all(workid string, writeLock bool) (err error) {
	if writeLock {
		err = cl.LockNamed("Delete_all")
		defer cl.Unlock()
	}

	cl._map = make(map[Workunit_Unique_Identifier]bool)
	cl.sync()

	return
}

func (cl *WorkunitList) Has(workid Workunit_Unique_Identifier) (ok bool, err error) {

	err = cl.LockNamed("Has")
	defer cl.Unlock()

	_, ok = cl._map[workid]

	return
}

func (cl *WorkunitList) Get_list(do_read_lock bool) (assigned_work_ids []Workunit_Unique_Identifier, err error) {
	if do_read_lock {
		read_lock, xerr := cl.RLockNamed("Get_list")
		if xerr != nil {
			err = xerr
			return
		}
		defer cl.RUnlockNamed(read_lock)
	}
	assigned_work_ids = []Workunit_Unique_Identifier{}
	for id, _ := range cl._map {

		assigned_work_ids = append(assigned_work_ids, id)
	}
	return
}

func (cl *WorkunitList) Get_string_list(do_read_lock bool) (work_ids []string, err error) {
	work_ids = []string{}
	if do_read_lock {
		read_lock, xerr := cl.RLockNamed("Get_string_list")
		if xerr != nil {
			err = xerr
			return
		}
		defer cl.RUnlockNamed(read_lock)
	}
	for id, _ := range cl._map {
		var work_str string
		work_str, err = id.String()
		if err != nil {
			err = fmt.Errorf("(Get_string_list) id.String() returned: %s", err.Error())
			return
		}
		work_ids = append(work_ids, work_str)
	}
	return
}

func (cl *WorkunitList) sync() (err error) {

	cl.Data = []string{}
	for id, _ := range cl._map {
		var work_str string
		work_str, err = id.String()
		if err != nil {
			err = fmt.Errorf("(sync) workid.String() returned: %s", err.Error())
			return
		}
		//id_string := id.String()

		cl.Data = append(cl.Data, work_str)
	}
	return
}

// opposite of sync; take Data entries and copy them into map
func (cl *WorkunitList) FillMap() (err error) {

	for _, id_str := range cl.Data {

		id, xerr := New_Workunit_Unique_Identifier_FromString(id_str)
		if xerr != nil {
			err = xerr
			return
		}

		cl._map[id] = true
	}

	return
}
