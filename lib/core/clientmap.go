package core

import (
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/rwmutex"
)

// ClientMap _
type ClientMap struct {
	rwmutex.RWMutex
	_map map[string]*Client
}

// NewClientMap _
func NewClientMap() *ClientMap {
	cm := &ClientMap{_map: make(map[string]*Client)}
	cm.RWMutex.Init("ClientMap")
	return cm
}

// Add _
func (cl *ClientMap) Add(client *Client, lock bool) (err error) {

	if lock {
		err = cl.LockNamed("(ClientMap) Add")
		if err != nil {
			return
		}
		defer cl.Unlock()
	}

	_, found := cl._map[client.ID]
	if found {
		logger.Warning("Client Id %s already exists.", client.ID)
	}

	cl._map[client.ID] = client

	return
}

// Get _
func (cl *ClientMap) Get(clientID string, lock bool) (client *Client, ok bool, err error) {

	if lock {
		readLock, xerr := cl.RLockNamed("Get")
		if xerr != nil {
			err = xerr
			return
		}
		defer cl.RUnlockNamed(readLock)
	}

	client, ok = cl._map[clientID]

	return
}

// Has _
func (cl *ClientMap) Has(clientID string, lock bool) (ok bool, err error) {

	if lock {
		readLock, xerr := cl.RLockNamed("Has")
		if xerr != nil {
			err = xerr
			return
		}
		defer cl.RUnlockNamed(readLock)
	}

	_, ok = cl._map[clientID]

	return
}

// Delete _
func (cl *ClientMap) Delete(clientID string, lock bool) (err error) {

	if lock {
		err = cl.LockNamed("(ClientMap) Delete")
		if err != nil {
			return
		}
	}
	delete(cl._map, clientID)
	if lock {
		cl.Unlock()
		logger.Debug(3, "(ClientMap) Delete done\n")
	}

	return
}

// GetClientIds _
func (cl *ClientMap) GetClientIds() (ids []string, err error) {

	readLock, err := cl.RLockNamed("GetClientIds")
	if err != nil {
		return
	}
	defer cl.RUnlockNamed(readLock)
	for id := range cl._map {
		ids = append(ids, id)
	}

	return
}

// GetClients _
func (cl *ClientMap) GetClients() (clients []*Client, err error) {

	clients = []*Client{}
	readLock, err := cl.RLockNamed("GetClients")
	if err != nil {
		return
	}
	defer cl.RUnlockNamed(readLock)
	for _, client := range cl._map {
		clients = append(clients, client)
	}

	return
}
