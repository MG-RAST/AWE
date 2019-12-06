package core

// Clients _
type Clients []*Client

// RLockRecursive _
func (cs *Clients) RLockRecursive() {
	for _, client := range *cs {
		client.RLockAnon()
	}
}

// RUnlockRecursive _
func (cs *Clients) RUnlockRecursive() {
	for _, client := range *cs {
		client.RUnlockAnon()
	}
}
