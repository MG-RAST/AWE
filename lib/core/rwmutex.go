package core

import (
	"github.com/MG-RAST/AWE/lib/core/uuid"
	"github.com/MG-RAST/AWE/lib/logger"
	//"strings"
	"sync"
	"time"
)

// This RWLock keeps track of the owener and of each reader
// Because channels are used for the locking mechanism, you can use locks with timeouts.
// Each reader gets a ReadLock object that has to be used to unlock after reading is done.

type ReadLock struct {
	id string
}

type RWMutex struct {
	//sync.RWMutex `bson:"-" json:"-"` // Locks only members that can change. Current_work has its own lock.
	writeLock chan int
	lockOwner string            `bson:"-" json:"-"`
	Name      string            `bson:"-" json:"-"`
	readers   map[string]string // map[Id]Name
	readLock  sync.Mutex
}

func (m *RWMutex) Init(name string) {
	m.Name = name
	m.writeLock = make(chan int, 1)
	m.readers = make(map[string]string)
	m.lockOwner = "nobody_init"

	if name == "" {
		panic("name not defined")
	}

	m.writeLock <- 1 // Put the initial value into the channel

	return
}

func (r *ReadLock) Get_Id() string {
	return r.id
}

func (m *RWMutex) Lock() {
	panic("Lock() was called")
	m.LockNamed("unknown")
}

func (m *RWMutex) LockNamed(name string) {
	//logger.Debug(3, "Lock")
	//reader_list := strings.Join(m.RList(), ",")
	//logger.Debug(3, "(%s) %s requests Lock. current owner:  %s (reader list :%s)", m.Name, name, m.lockOwner, reader_list)
	if m.Name == "" {
		panic("LockNamed: object has no name")
	}
	//m.RWMutex.Lock()
	<-m.writeLock // Grab the ticket
	m.lockOwner = name

	// wait for Readers to leave....

	for m.RCount() > 0 {
		time.Sleep(100)
	}

	//logger.Debug(3, "(%s) LOCKED by %s", m.Name, name)
}

func (m *RWMutex) Unlock() {
	//logger.Debug(3, "Unlock")
	old_owner := m.lockOwner
	m.lockOwner = "nobody_anymore"
	//m.RWMutex.Unlock()
	m.writeLock <- 1 // Give it back
	logger.Debug(3, "(%s) UNLOCKED by %s **********************", m.Name, old_owner)
}

func (m *RWMutex) RLockNamed(name string) ReadLock {
	logger.Debug(3, "(%s) request RLock and Lock.", m.Name)
	if m.Name == "" {
		panic("xzy name empty")
	}
	m.LockNamed("RLock")
	logger.Debug(3, "(%s) RLock got Lock.", m.Name)
	m.readLock.Lock()
	new_uuid := uuid.New()

	m.readers[new_uuid] = name

	//m.readCounter += 1
	m.readLock.Unlock()
	m.Unlock()
	logger.Debug(3, "(%s) got RLock.", m.Name)
	return ReadLock{id: new_uuid}
}

func (m *RWMutex) RLock() {
	panic("RLock() was called")
}
func (m *RWMutex) RUnlock() {
	panic("RUnlock() was called")
}

func (m *RWMutex) RUnlockNamed(rl ReadLock) {

	lock_uuid := rl.Get_Id()

	logger.Debug(3, "(%s) request RUnlock.", m.Name)
	m.readLock.Lock()
	//m.readCounter -= 1
	name, ok := m.readers[lock_uuid]
	if ok {
		delete(m.readers, lock_uuid)
		logger.Debug(3, "(%s) %s did RUnlock.", m.Name, name)
	} else {
		logger.Debug(3, "(%s) ERROR: %s did not have RLock !?!??!?!?.", m.Name, name)
	}
	m.readLock.Unlock()
	logger.Debug(3, "(%s) did RUnlock.", m.Name)
}

func (m *RWMutex) RCount() (c int) {

	m.readLock.Lock()
	c = len(m.readers)
	m.readLock.Unlock()
	return
}

func (m *RWMutex) RList() (list []string) {

	m.readLock.Lock()
	for _, v := range m.readers {
		list = append(list, v)
	}
	m.readLock.Unlock()
	return
}
