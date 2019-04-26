// +build !race

package rwmutex

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/MG-RAST/AWE/lib/core/uuid"
	"github.com/MG-RAST/AWE/lib/logger"
)

// This RWLock keeps track of the owener and of each reader
// Because channels are used for the locking mechanism, you can use locks with timeouts.
// Each reader gets a ReadLock object that has to be used to unlock after reading is done.

var WAIT_TIMEOUT = time.Minute * 3

var DEBUG_LOCKS = false

type ReadLock struct {
	id string
}

type RWMutex struct {
	//sync.RWMutex `bson:"-" json:"-"` // Locks only members that can change. Current_work has its own lock.
	writeLock   chan int
	lockOwner   StringLocked
	Name        string            `bson:"-" json:"-"`
	readers     map[string]string // map[Id]Name (named readers)
	anonCounter int               // (anonymous readers, use only when necessary)
	readLock    sync.Mutex
}

type StringLocked struct {
	sync.Mutex
	value string
}

func (s *StringLocked) Set(value string) {
	s.Lock()
	defer s.Unlock()
	s.value = value
}

func (s *StringLocked) Get() string {
	s.Lock()
	defer s.Unlock()
	return s.value
}

func (m *RWMutex) Init(name string) {
	m.Name = name
	m.writeLock = make(chan int, 1)
	m.readers = make(map[string]string)
	m.anonCounter = 0

	m.lockOwner.Set("nobody_init")

	if name == "" {
		panic("name not defined")
	}

	m.writeLock <- 1 // Put the initial value into the channel

	return
}

func (r *ReadLock) Get_Id() string {
	return r.id
}

func (m *RWMutex) Lock() (err error) {
	panic("Lock() was called")
	//err = m.LockNamed("unknown")
	//return
}

func (m *RWMutex) LockNamed(name string) (err error) {
	//logger.Debug(3, "Lock")
	//reader_list := strings.Join(m.RList(), ",")
	//logger.Debug(3, "(%s) %s requests Lock. current owner:  %s (reader list :%s)", m.Name, name, m.lockOwner.Get(), reader_list) // reading  m.lockOwner induces data race !
	if m.Name == "" {
		panic("LockNamed: object has no name")
	}
	initial_owner := m.lockOwner.Get()

	select {
	case <-m.writeLock: // Grab the ticket
		if DEBUG_LOCKS {
			logger.Debug(3, "(RWMutex/LockNamed %s) got lock", name)
		}
	case <-time.After(WAIT_TIMEOUT):
		reader_list := strings.Join(m.RList(), ",")
		message := fmt.Sprintf("(%s) %s requests Lock. TIMEOUT!!! current owner: %s (reader list :%s), initial owner: %s", m.Name, name, m.lockOwner.Get(), reader_list, initial_owner)
		logger.Error(message)
		err = fmt.Errorf(message)
		return
	}

	m.lockOwner.Set(name)

	// wait for Readers to leave....
	for m.RCount() > 0 {
		time.Sleep(100)
	}

	if DEBUG_LOCKS {
		logger.Debug(3, "(%s) LOCKED by %s", m.Name, name)
	}
	return
}

func (m *RWMutex) Unlock() {
	//logger.Debug(3, "Unlock")
	old_owner := m.lockOwner.Get()
	m.lockOwner.Set("nobody_anymore")
	m.writeLock <- 1 // Give it back
	if DEBUG_LOCKS {
		logger.Debug(3, "(%s) UNLOCKED by %s **********************", m.Name, old_owner)
	}
}

func (m *RWMutex) RLockNamed(name string) (rl ReadLock, err error) {
	if DEBUG_LOCKS {
		logger.Debug(3, "(%s, %s) request RLock and Lock.", m.Name, name)
	}
	if m.Name == "" {
		panic("xzy name empty")
	}
	err = m.LockNamed(m.Name + "/RLock " + name)
	if err != nil {
		return
	}
	if DEBUG_LOCKS {
		logger.Debug(3, "(%s) RLock/%s got Lock.", m.Name, name)
	}
	m.readLock.Lock()
	new_uuid := uuid.New()

	m.readers[new_uuid] = name

	//m.readCounter += 1
	m.readLock.Unlock()
	m.Unlock()
	if DEBUG_LOCKS {
		logger.Debug(3, "(%s) got RLock.", m.Name)
	}
	rl = ReadLock{id: new_uuid}
	return
}

func (m *RWMutex) RLock() {
	panic("RLock() was called")
}
func (m *RWMutex) RUnlock() {
	panic("RUnlock() was called")
}

func (m *RWMutex) RLockAnon() {
	m.readLock.Lock()
	defer m.readLock.Unlock()
	m.anonCounter += 1
}
func (m *RWMutex) RUnlockAnon() {
	m.readLock.Lock()
	defer m.readLock.Unlock()
	m.anonCounter -= 1
}

func (m *RWMutex) RUnlockNamed(rl ReadLock) {

	lock_uuid := rl.Get_Id()

	if DEBUG_LOCKS {
		logger.Debug(3, "(%s) request RUnlock.", m.Name)
	}
	m.readLock.Lock()
	//m.readCounter -= 1
	name, ok := m.readers[lock_uuid]
	if ok {
		delete(m.readers, lock_uuid)
		if DEBUG_LOCKS {
			logger.Debug(3, "(%s) %s did RUnlock.", m.Name, name)
		}
	} else {
		logger.Debug(3, "(%s) ERROR: %s did not have RLock !?!??!?!?.", m.Name, name)
	}
	m.readLock.Unlock()
	if DEBUG_LOCKS {
		logger.Debug(3, "(%s) did RUnlock.", m.Name)
	}
}

func (m *RWMutex) RCount() (c int) {

	m.readLock.Lock()
	c = len(m.readers)
	c += m.anonCounter
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
