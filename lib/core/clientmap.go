package core

import (
	"fmt"
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

func (cl *ClientMap) GetMap() *map[string]*Client {
	fmt.Println("(ClientMap) GetMap\n")
	return &cl._map
}

func (cl *ClientMap) Add(client *Client, lock bool) {

	if lock {
		cl.LockNamed("(ClientMap) Add")
	}
	cl._map[client.Id] = client
	if lock {
		cl.Unlock()
	}

}

func (cl *ClientMap) Get(client_id string, lock bool) (client *Client, ok bool) {

	if lock {
		read_lock := cl.RLockNamed("Get")
		defer cl.RUnlockNamed(read_lock)
	}

	client, ok = cl._map[client_id]

	return
}

func (cl *ClientMap) Delete(client_id string, lock bool) {

	if lock {
		cl.LockNamed("(ClientMap) Delete")
	}
	delete(cl._map, client_id)
	if lock {
		cl.Unlock()
		fmt.Println("(ClientMap) Delete done\n")
	}

	return
}

func (cl *ClientMap) GetClientIds() (ids []string) {

	read_lock := cl.RLockNamed("GetClientIds")
	defer cl.RUnlockNamed(read_lock)
	for id, _ := range cl._map {
		ids = append(ids, id)
	}

	return
}

func (cl *ClientMap) GetClients() (clients []*Client) {

	read_lock := cl.RLockNamed("GetClients")
	defer cl.RUnlockNamed(read_lock)
	for _, client := range cl._map {
		clients = append(clients, client)
	}

	return
}
