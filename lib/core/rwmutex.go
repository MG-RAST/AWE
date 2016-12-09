package core

import (
	"fmt"
	//"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/core/uuid"
	"strings"
	"sync"
	"time"
)

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
	fmt.Printf("(%s) created\n", name)

	m.Name = name
	m.writeLock = make(chan int, 1)
	m.readers = make(map[string]string)
	m.lockOwner = "nobody_init"

	if name == "" {
		panic("name not defined")
	}

	m.writeLock <- 1 // Put the initial value into the channel
	fmt.Printf("Init. current name:  %s\n", m.Name)
	fmt.Printf("Init. current owner:  %s\n", m.lockOwner)
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
	reader_list := strings.Join(m.RList(), ",")
	fmt.Printf("(%s) %s requests Lock. current owner:  %s (reader list :%s)\n", m.Name, name, m.lockOwner, reader_list)
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

	fmt.Printf("(%s) LOCKED by %s. \n", m.Name, name)
}

func (m *RWMutex) Unlock() {
	//logger.Debug(3, "Unlock")
	old_owner := m.lockOwner
	m.lockOwner = "nobody_anymore"
	//m.RWMutex.Unlock()
	m.writeLock <- 1 // Give it back
	fmt.Printf("(%s) UNLOCKED by %s **********************\n", m.Name, old_owner)
}

func (m *RWMutex) RLockNamed(name string) ReadLock {
	fmt.Printf("(%s) request RLock and Lock.\n", m.Name)
	if m.Name == "" {
		panic("xzy name empty")
	}
	m.LockNamed("RLock")
	fmt.Printf("(%s) RLock got Lock.\n", m.Name)
	m.readLock.Lock()
	new_uuid := uuid.New()

	m.readers[new_uuid] = name

	//m.readCounter += 1
	m.readLock.Unlock()
	m.Unlock()
	fmt.Printf("(%s) got RLock.\n", m.Name)
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

	fmt.Printf("(%s) request RUnlock.\n", m.Name)
	m.readLock.Lock()
	//m.readCounter -= 1
	name, ok := m.readers[lock_uuid]
	if ok {
		delete(m.readers, lock_uuid)
		fmt.Printf("(%s) %s did RUnlock.\n", m.Name, name)
	} else {
		fmt.Printf("(%s) ERROR: %s did not have RLock !?!??!?!?.\n", m.Name, name)
	}
	m.readLock.Unlock()
	fmt.Printf("(%s) did RUnlock.\n", m.Name)
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
