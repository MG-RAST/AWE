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
	cm.RWMutex.Name = "ClientMap"
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

func (cl *ClientMap) Get(client_id string) (client *Client, ok bool) {
	fmt.Println("(ClientMap) Get\n")
	cl.RLock()
	client, ok = cl._map[client_id]
	cl.RUnlock()
	fmt.Println("(ClientMap) Get done\n")
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
	fmt.Println("(GetClientIds) GetClientIds\n")
	cl.RLock()
	for id, _ := range cl._map {
		ids = append(ids, id)
	}
	cl.RUnlock()
	fmt.Println("(GetClientIds) GetClientIds done\n")
	return
}
