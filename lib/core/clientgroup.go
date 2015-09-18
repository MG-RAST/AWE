package core

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/clientGroupAcl"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core/uuid"
	"github.com/MG-RAST/AWE/lib/user"
	"github.com/MG-RAST/AWE/vendor/github.com/MG-RAST/golib/uniuri"
	"github.com/MG-RAST/AWE/vendor/gopkg.in/mgo.v2/bson"
	"os"
	"regexp"
	"strconv"
	"time"
)

type ClientGroup struct {
	Id           string                        `bson:"id" json:"id"`
	IP_CIDR      string                        `bson:"ip_cidr" json:"ip_cidr"`
	Name         string                        `bson:"name" json:"name"`
	Token        string                        `bson:"token" json:"token"`
	Acl          clientGroupAcl.ClientGroupAcl `bson:"acl" json:"-"`
	CreatedOn    time.Time                     `bson:"created_on" json:"created_on"`
	Expiration   time.Time                     `bson:"expiration" json:"expiration"`
	LastModified time.Time                     `bson:"last_modified" json:"last_modified"`
}

var (
	CGNameRegex = regexp.MustCompile(`^[A-Za-z0-9\_\-\.]+$`)
)

func CreateClientGroup(name string, u *user.User) (cg *ClientGroup, err error) {
	q := bson.M{"name": name}
	clientgroups := new(ClientGroups)
	if count, err := dbFindClientGroups(q, clientgroups); err != nil {
		return nil, err
	} else if count > 0 {
		return nil, errors.New("A clientgroup already exists with this name.")
	}

	if !CGNameRegex.Match([]byte(name)) {
		return nil, errors.New(fmt.Sprintf("Clientgroup name (%s) must contain only alphanumeric characters, underscore, dash or dot.", name))
	}

	t := time.Now()

	cg = new(ClientGroup)
	cg.Id = uuid.New()
	cg.IP_CIDR = "0.0.0.0/0"
	cg.Name = name
	cg.Expiration = t.AddDate(10, 0, 0)
	cg.Acl.SetOwner(u.Uuid)
	cg.Acl.Set(u.Uuid, clientGroupAcl.Rights{"read": true, "write": true, "delete": true, "execute": true})
	cg.CreatedOn = t
	cg.SetToken()

	if err = cg.Save(); err != nil {
		err = errors.New("error in cg.Save(), error=" + err.Error())
		return
	}

	return
}

func (cg *ClientGroup) SetToken() {
	var host string
	if hostname, err := os.Hostname(); err == nil {
		host = fmt.Sprintf("%s:%d", hostname, conf.API_PORT)
	}
	cg.Token = "name=" + cg.Name + "|id=" + cg.Id + "|exp=" + strconv.FormatInt(cg.Expiration.Unix(), 10) + "|server=" + host + "|sig=" + uniuri.NewLen(256)
	return
}

func (cg *ClientGroup) Save() (err error) {
	cg.LastModified = time.Now()
	err = dbUpsert(cg)
	return
}
