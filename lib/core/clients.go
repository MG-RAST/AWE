package core

type Clients []*Client

func (cs *Clients) RLockRecursive() {
	for _, client := range *cs {
		client.RLockAnon()
	}
}

func (cs *Clients) RUnlockRecursive() {
	for _, client := range *cs {
		client.RUnlockAnon()
	}
}
