package clientGroupAcl

import ()

// Acl struct
type ClientGroupAcl struct {
	Owner   string   `bson:"owner" json:"owner"`
	Read    []string `bson:"read" json:"read"`
	Write   []string `bson:"write" json:"write"`
	Delete  []string `bson:"delete" json:"delete"`
	Execute []string `bson:"execute" json:"execute"`
}

type Rights map[string]bool

func (a *ClientGroupAcl) SetOwner(str string) {
	a.Owner = str
	return
}

func (a *ClientGroupAcl) UnSet(str string, r Rights) {
	if r["read"] {
		a.Read = del(a.Read, str)
	}
	if r["write"] {
		a.Write = del(a.Write, str)
	}
	if r["delete"] {
		a.Delete = del(a.Delete, str)
	}
	if r["execute"] {
		a.Execute = del(a.Execute, str)
	}
	return
}

func (a *ClientGroupAcl) Set(str string, r Rights) {
	if r["read"] {
		a.Read = insert(a.Read, str)
	}
	if r["write"] {
		a.Write = insert(a.Write, str)
	}
	if r["delete"] {
		a.Delete = insert(a.Delete, str)
	}
	if r["execute"] {
		a.Execute = insert(a.Execute, str)
	}
	return
}

func (a *ClientGroupAcl) Check(str string) (r Rights) {
	r = Rights{"read": false, "write": false, "delete": false, "execute": false}
	acls := map[string][]string{"read": a.Read, "write": a.Write, "delete": a.Delete, "execute": a.Execute}
	for k, v := range acls {
		if len(v) == 0 {
			r[k] = true
		} else {
			for _, id := range v {
				if str == id {
					r[k] = true
					break
				}
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
