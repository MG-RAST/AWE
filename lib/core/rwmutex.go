package core

import (
	"fmt"
	"github.com/MG-RAST/AWE/lib/logger"
	"sync"
)

type RWMutex struct {
	sync.RWMutex `bson:"-" json:"-"` // Locks only members that can change. Current_work has its own lock.
	LockOwner    string              `bson:"-" json:"-"`
	Name         string              `bson:"-" json:"-"`
}

func (m *RWMutex) Lock() {
	logger.Debug(3, "Lock")
	fmt.Printf("request Lock ............... current owner: %s\n", m.LockOwner)
	m.RWMutex.Lock()
	m.LockOwner = "unknown"
	fmt.Printf("LOCKED by unknown **********************\n")
}

func (m *RWMutex) LockNamed(name string) {
	logger.Debug(3, "Lock")
	fmt.Printf("(%s) request Lock ............... current owner:  %s\n", m.Name, m.LockOwner)
	m.RWMutex.Lock()
	m.LockOwner = name
	fmt.Printf("(%s) LOCKED by %s ********************** \n", m.Name, name)
}

func (m *RWMutex) Unlock() {
	logger.Debug(3, "Unlock")
	old_owner := m.LockOwner
	m.LockOwner = "nobody"
	m.RWMutex.Unlock()
	fmt.Printf("(%s) UNLOCKED by %s **********************\n", m.Name, old_owner)
}
