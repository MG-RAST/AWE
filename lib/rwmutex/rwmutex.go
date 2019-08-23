// +build !race

package rwmutex

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/MG-RAST/AWE/lib/logger"
	uuid "github.com/MG-RAST/golib/go-uuid/uuid"
)

// This RWLock keeps track of the owener and of each reader
// Because channels are used for the locking mechanism, you can use locks with timeouts.
// Each reader gets a ReadLock object that has to be used to unlock after reading is done.

// WaitTimeout _
var WaitTimeout = time.Second * 10 //time.Minute * 3

var debugLocks = false

// ReadLock _
type ReadLock struct {
	id string
}

// RWMutex _
type RWMutex struct {
	//sync.RWMutex `bson:"-" json:"-"` // Locks only members that can change. Current_work has its own lock.
	writeLock   chan int
	lockOwner   StringLocked
	Name        string            `bson:"-" json:"-"`
	readers     map[string]string // map[Id]Name (named readers)
	anonCounter int               // (anonymous readers, use only when necessary)
	readLock    sync.Mutex
}

// StringLocked _
type StringLocked struct {
	sync.Mutex
	value string
}

// Set _
func (s *StringLocked) Set(value string) {
	s.Lock()
	defer s.Unlock()
	s.value = value
}

// Get _
func (s *StringLocked) Get() string {
	s.Lock()
	defer s.Unlock()
	return s.value
}

// Init _
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

// GetID _
func (r *ReadLock) GetID() string {
	return r.id
}

// Lock _
func (m *RWMutex) Lock() (err error) {
	panic("Lock() was called")
	//err = m.LockNamed("unknown")
	//return
}

// LockNamedTimeout _
func (m *RWMutex) LockNamedTimeout(name string, timeout time.Duration) (err error) {

	//logger.Debug(3, "Lock")
	//reader_list := strings.Join(m.RList(), ",")
	//logger.Debug(3, "(%s) %s requests Lock. current owner:  %s (reader list :%s)", m.Name, name, m.lockOwner.Get(), reader_list) // reading  m.lockOwner induces data race !

	if m.Name == "" {
		panic("LockNamed: object has no name")
	}
	initialOwner := m.lockOwner.Get()

	select {
	case <-m.writeLock: // Grab the ticket
		if debugLocks {
			logger.Debug(3, "(RWMutex/LockNamed %s) got lock", name)
		}
	case <-time.After(timeout):
		readerList := strings.Join(m.RList(), ",")
		message := fmt.Sprintf("(%s) %s requests Lock. TIMEOUT!!! (%s) current owner: %s (reader list :%s), initial owner: %s", m.Name, name, timeout, m.lockOwner.Get(), readerList, initialOwner)
		//logger.Error(message)
		err = fmt.Errorf(message)
		return
	}

	m.lockOwner.Set(name)

	// wait for Readers to leave....
	for m.RCount() > 0 {
		time.Sleep(100)
	}

	if debugLocks {
		logger.Debug(3, "(%s) LOCKED by %s", m.Name, name)
	}
	return

}

// LockNamed _
func (m *RWMutex) LockNamed(name string) (err error) {
	err = m.LockNamedTimeout(name, WaitTimeout)
	return
}

// Unlock _
func (m *RWMutex) Unlock() {
	//logger.Debug(3, "Unlock")
	oldOwner := m.lockOwner.Get()
	m.lockOwner.Set("nobody_anymore")
	m.writeLock <- 1 // Give it back
	if debugLocks {
		logger.Debug(3, "(%s) UNLOCKED by %s **********************", m.Name, oldOwner)
	}
}

// RLockNamedTimeout _
func (m *RWMutex) RLockNamedTimeout(name string, timeout time.Duration) (rl ReadLock, err error) {
	if debugLocks {
		logger.Debug(3, "(%s, %s) request RLock and Lock.", m.Name, name)
	}
	if m.Name == "" {
		panic("xzy name empty")
	}
	err = m.LockNamedTimeout(m.Name+"/RLock "+name, timeout)
	if err != nil {
		return
	}
	if debugLocks {
		logger.Debug(3, "(%s) RLock/%s got Lock.", m.Name, name)
	}
	m.readLock.Lock()
	newUUID := uuid.New()

	m.readers[newUUID] = name

	//m.readCounter += 1
	m.readLock.Unlock()
	m.Unlock()
	if debugLocks {
		logger.Debug(3, "(%s) got RLock.", m.Name)
	}
	rl = ReadLock{id: newUUID}
	return
}

// RLockNamed _
func (m *RWMutex) RLockNamed(name string) (rl ReadLock, err error) {
	rl, err = m.RLockNamedTimeout(name, WaitTimeout)
	return
}

// RLock _
func (m *RWMutex) RLock() {
	panic("RLock() was called")
}

// RUnlock _
func (m *RWMutex) RUnlock() {
	panic("RUnlock() was called")
}

// RLockAnon _
func (m *RWMutex) RLockAnon() {
	m.readLock.Lock()
	defer m.readLock.Unlock()
	m.anonCounter++
}

// RUnlockAnon _
func (m *RWMutex) RUnlockAnon() {
	m.readLock.Lock()
	defer m.readLock.Unlock()
	m.anonCounter--
}

// RUnlockNamed _
func (m *RWMutex) RUnlockNamed(rl ReadLock) {

	lockUUID := rl.GetID()

	if debugLocks {
		logger.Debug(3, "(%s) request RUnlock.", m.Name)
	}
	m.readLock.Lock()
	//m.readCounter -= 1
	name, ok := m.readers[lockUUID]
	if ok {
		delete(m.readers, lockUUID)
		if debugLocks {
			logger.Debug(3, "(%s) %s did RUnlock.", m.Name, name)
		}
	} else {
		logger.Debug(3, "(%s) ERROR: %s did not have RLock !?!??!?!?.", m.Name, name)
	}
	m.readLock.Unlock()
	if debugLocks {
		logger.Debug(3, "(%s) did RUnlock.", m.Name)
	}
}

// RCount _
func (m *RWMutex) RCount() (c int) {

	m.readLock.Lock()
	c = len(m.readers)
	c += m.anonCounter
	m.readLock.Unlock()
	return
}

// RList _
func (m *RWMutex) RList() (list []string) {

	m.readLock.Lock()
	for _, v := range m.readers {
		list = append(list, v)
	}
	m.readLock.Unlock()
	return
}
