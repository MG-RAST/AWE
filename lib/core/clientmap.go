package core

import (
	"github.com/MG-RAST/AWE/lib/logger"
)

type ClientMap struct {
	RWMutex
	_map map[string]*Client
}

func NewClientMap() *ClientMap {
	cm := &ClientMap{_map: make(map[string]*Client)}
	cm.RWMutex.Init("ClientMap")
	return cm
}

func (cl *ClientMap) Add(client *Client, lock bool) {

	if lock {
		cl.LockNamed("(ClientMap) Add")
		defer cl.Unlock()
	}

	_, found := cl._map[client.Id]
	if found {
		log.Warn("Client Id % already exists.", client.Id)
	}

	cl._map[client.Id] = client

	return
}

func (cl *ClientMap) Get(client_id string, lock bool) (client *Client, ok bool, err error) {

	if lock {
		read_lock, xerr := cl.RLockNamed("Get")
		if xerr != nil {
			err = xerr
			return
		}
		defer cl.RUnlockNamed(read_lock)
	}

	client, ok = cl._map[client_id]

	return
}

func (cl *ClientMap) Delete(client_id string, lock bool) (err error) {

	if lock {
		err = cl.LockNamed("(ClientMap) Delete")
		if err != nil {
			return
		}
	}
	delete(cl._map, client_id)
	if lock {
		cl.Unlock()
		logger.Debug(3, "(ClientMap) Delete done\n")
	}

	return
}

func (cl *ClientMap) GetClientIds() (ids []string, err error) {

	read_lock, err := cl.RLockNamed("GetClientIds")
	if err != nil {
		return
	}
	defer cl.RUnlockNamed(read_lock)
	for id, _ := range cl._map {
		ids = append(ids, id)
	}

	return
}

func (cl *ClientMap) GetClients() (clients []*Client, err error) {

	clients = []*Client{}
	read_lock, err := cl.RLockNamed("GetClients")
	if err != nil {
		return
	}
	defer cl.RUnlockNamed(read_lock)
	for _, client := range cl._map {
		clients = append(clients, client)
	}

	return
}
