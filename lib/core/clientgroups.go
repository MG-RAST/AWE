package core

import (
	"gopkg.in/mgo.v2/bson"
)

// ClientGroup array type
type ClientGroups []ClientGroup

func (n *ClientGroups) GetPaginated(q bson.M, limit int, offset int, order string, direction string) (count int, err error) {
	if direction == "desc" {
		order = "-" + order
	}
	count, err = dbFindSortClientGroups(q, n, map[string]int{"limit": limit, "offset": offset}, order)
	return
}
