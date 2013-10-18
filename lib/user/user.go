package user

import (
	"code.google.com/p/go-uuid/uuid"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/db"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

// Array of User
type Users []User

// User struct
type User struct {
	Uuid         string      `bson:"uuid" json:"uuid"`
	Username     string      `bson:"username" json:"username"`
	Fullname     string      `bson:"fullname" json:"fullname"`
	Email        string      `bson:"email" json:"email"`
	Password     string      `bson:"password" json:"-"`
	Admin        bool        `bson:"admin" json:"admin"`
	CustomFields interface{} `bson:"custom_fields" json:"custom_fields"`
}

func Initialize() {
	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C("Users")
	c.EnsureIndex(mgo.Index{Key: []string{"uuid"}, Unique: true})
	c.EnsureIndex(mgo.Index{Key: []string{"username"}, Unique: true})
	New("public", "public", false) //initialize a default public user
}

func New(username string, password string, isAdmin bool) (u *User, err error) {
	u = &User{Uuid: uuid.New(), Username: username, Password: password, Admin: isAdmin}
	if err = u.Save(); err != nil {
		u = nil
	}
	return
}

func FindByUuid(uuid string) (u *User, err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C("Users")
	u = &User{Uuid: uuid}
	if err = c.Find(bson.M{"uuid": u.Uuid}).One(&u); err != nil {
		return nil, err
	}
	return
}

func FindByUsernamePassword(username string, password string) (u *User, err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C("Users")
	u = &User{}
	if err = c.Find(bson.M{"username": username, "password": password}).One(&u); err != nil {
		return nil, err
	}
	return
}

func AdminGet(u *Users) (err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C("Users")
	err = c.Find(nil).All(u)
	return
}

func (u *User) SetUuid() (err error) {
	if uu, err := dbGetUuid(u.Email); err == nil {
		u.Uuid = uu
		return nil
	} else {
		u.Uuid = uuid.New()
		if err := u.Save(); err != nil {
			return err
		}
	}
	return
}

func dbGetUuid(email string) (uuid string, err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C("Users")
	u := User{}
	if err = c.Find(bson.M{"email": email}).One(&u); err != nil {
		return "", err
	}
	return u.Uuid, nil
}

func (u *User) Save() (err error) {
	session := db.Connection.Session.Copy()
	defer session.Close()
	c := session.DB(conf.MONGODB_DATABASE).C("Users")
	err = c.Insert(&u)
	return
}

func (u *User) RemovePasswd() (nu *User) {
	nu = &User{Uuid: u.Uuid, Username: u.Username, Password: "**********", Admin: u.Admin}
	return
}
