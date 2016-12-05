package core

import (
	"sync"
)

type ClientMap struct {
	sync.RWMutex
	_map map[string]*Client
}

func NewClientMap() *ClientMap {
	return &ClientMap{_map: make(map[string]*Client)}
}

func (cl *ClientMap) GetMap() *map[string]*Client {
	return &cl._map
}

func (cl *ClientMap) Add(client *Client, lock bool) {
	if lock {
		cl.Lock()
	}
	cl._map[client.Id] = client
	if lock {
		cl.Unlock()
	}
}

func (cl *ClientMap) Get(client_id string) (client *Client, ok bool) {
	cl.RLock()
	client, ok = cl._map[client_id]
	cl.RUnlock()
	return
}

func (cl *ClientMap) Delete(client_id string, lock bool) {
	if lock {
		cl.Lock()
	}
	delete(cl._map, client_id)
	if lock {
		cl.Unlock()
	}
	return
}

func (cl *ClientMap) GetClientIds() (ids []string) {
	cl.RLock()
	for id, _ := range cl._map {
		ids = append(ids, id)
	}
	cl.RUnlock()
	return
}
