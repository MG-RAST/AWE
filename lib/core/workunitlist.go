package core

type WorkunitList struct {
	RWMutex `bson:"-" json:"-"`
	_map    map[Workunit_Unique_Identifier]bool `json:"-"`
	Data    []string                            `json:"data"`
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

func (cl *WorkunitList) Delete(workid Workunit_Unique_Identifier, write_lock bool) (err error) {
	if write_lock {
		err = cl.LockNamed("Delete")
		defer cl.Unlock()
	}
	delete(cl._map, workid)
	cl.sync()
	return
}

func (cl *WorkunitList) Delete_all(workid string, write_lock bool) (err error) {
	if write_lock {
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

		work_ids = append(work_ids, id.String())
	}
	return
}

func (cl *WorkunitList) sync() (err error) {

	cl.Data = []string{}
	for id, _ := range cl._map {

		id_string := id.String()

		cl.Data = append(cl.Data, id_string)
	}
	return
}
