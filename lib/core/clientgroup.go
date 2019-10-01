package core

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/MG-RAST/AWE/lib/clientGroupAcl"
	"github.com/MG-RAST/AWE/lib/conf"
	uuid "github.com/MG-RAST/golib/go-uuid/uuid"
	"github.com/MG-RAST/AWE/lib/user"
	"github.com/MG-RAST/golib/uniuri"
	"gopkg.in/mgo.v2/bson"
)

// ClientGroup _
type ClientGroup struct {
	ID           string                        `bson:"id" json:"id"`
	IPCidr       string                        `bson:"ip_cidr" json:"ip_cidr"`
	Name         string                        `bson:"name" json:"name"`
	Token        string                        `bson:"token" json:"token"`
	ACL          clientGroupAcl.ClientGroupAcl `bson:"acl" json:"-"`
	CreatedOn    time.Time                     `bson:"created_on" json:"created_on"`
	Expiration   time.Time                     `bson:"expiration" json:"expiration"`
	LastModified time.Time                     `bson:"last_modified" json:"last_modified"`
}

var (
	// CGNameRegex _
	CGNameRegex = regexp.MustCompile(`^[A-Za-z0-9\_\-\.]+$`)
)

// CreateClientGroup _
func CreateClientGroup(name string, u *user.User) (cg *ClientGroup, err error) {
	q := bson.M{"name": name}
	clientgroups := new(ClientGroups)
	var count int
	if count, err = dbFindClientGroups(q, clientgroups); err != nil {
		err = fmt.Errorf("(CreateClientGroup) dbFindClientGroups returned: %s", err.Error())
		return
	} else if count > 0 {
		err = errors.New("a clientgroup already exists with this name")
		return
	}

	if !CGNameRegex.Match([]byte(name)) {
		err = fmt.Errorf("Clientgroup name (%s) must contain only alphanumeric characters, underscore, dash or dot", name)
		return
	}

	t := time.Now()

	cg = new(ClientGroup)
	cg.ID = uuid.New()
	cg.IPCidr = "0.0.0.0/0"
	cg.Name = name
	cg.Expiration = t.AddDate(10, 0, 0)
	cg.ACL.SetOwner(u.Uuid)
	cg.ACL.Set(u.Uuid, clientGroupAcl.Rights{"read": true, "write": true, "delete": true, "execute": true})
	cg.CreatedOn = t
	cg.SetToken()

	if err = cg.Save(); err != nil {
		err = errors.New("error in cg.Save(), error=" + err.Error())
		return
	}

	return
}

// SetToken _
func (cg *ClientGroup) SetToken() {
	var host string
	if hostname, err := os.Hostname(); err == nil {
		host = fmt.Sprintf("%s:%d", hostname, conf.API_PORT)
	}
	cg.Token = "name=" + cg.Name + "|id=" + cg.ID + "|exp=" + strconv.FormatInt(cg.Expiration.Unix(), 10) + "|server=" + host + "|sig=" + uniuri.NewLen(256)
	return
}

// Save _
func (cg *ClientGroup) Save() (err error) {
	cg.LastModified = time.Now()
	err = dbUpsert(cg)
	return
}
