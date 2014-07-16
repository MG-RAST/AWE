package acl

import ()

// Acl struct
type Acl struct {
	Owner  string   `bson:"owner" json:"owner"`
	Read   []string `bson:"read" json:"read"`
	Write  []string `bson:"write" json:"write"`
	Delete []string `bson:"delete" json:"delete"`
}

type Rights map[string]bool

func (a *Acl) SetOwner(str string) {
	a.Owner = str
	return
}

func (a *Acl) UnSet(str string, r Rights) {
	if r["read"] {
		a.Read = del(a.Read, str)
	}
	if r["write"] {
		a.Write = del(a.Write, str)
	}
	if r["delete"] {
		a.Delete = del(a.Delete, str)
	}
	return
}

func (a *Acl) Set(str string, r Rights) {
	if r["read"] {
		a.Read = insert(a.Read, str)
	}
	if r["write"] {
		a.Write = insert(a.Write, str)
	}
	if r["delete"] {
		a.Delete = insert(a.Delete, str)
	}
	return
}

func (a *Acl) Check(str string) (r Rights) {
	r = Rights{"read": false, "write": false, "delete": false}
	acls := map[string][]string{"read": a.Read, "write": a.Write, "delete": a.Delete}
	for k, v := range acls {
		for _, id := range v {
			if str == id {
				r[k] = true
				break
			}
		}
	}
	return
}

func del(arr []string, s string) (narr []string) {
	narr = []string{}
	for i, item := range arr {
		if item != s {
			narr = append(narr, item)
		} else {
			narr = append(narr, arr[i+1:]...)
			break
		}
	}
	return
}

func insert(arr []string, s string) []string {
	for _, item := range arr {
		if item == s {
			return arr
		}
	}
	return append(arr, s)
}
