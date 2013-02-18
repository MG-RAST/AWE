package core

import (
	"encoding/json"
	"github.com/MG-RAST/AWE/core/uuid"
	"io/ioutil"
	"time"
)

type Client struct {
	Id         string    `bson:"id" json:"id"`
	Name       string    `bson:"name" json:"name"`
	Group      string    `bson:"group" json:"group"`
	User       string    `bson:"user" json:"user"`
	Host       string    `bson:"host" json:"host"`
	Workers    int       `bson:"workers" json:"workers"`
	LocalShock string    `bson:"localshock" json:localshock"`
	Cores      int       `bson:"cores" json:"cores"`
	Apps       []string  `bson:"apps" json:"apps"`
	SkipWorks  []string  `bson:"SkipWorkunits" json:"SkipWorkunits"`
	RegTime    time.Time `bson:"regtime" json:"regtime"`
	Status     string    `bson:"Status" json:"Status"`
	Tag        bool      `bson:"-" json:"-"`
}

func NewClient() (client *Client) {
	client = new(Client)
	client.Id = uuid.New()
	client.Apps = []string{}
	client.SkipWorks = []string{}
	client.Status = "active"
	client.Tag = true
	return
}

func NewProfileClient(filepath string) (client *Client, err error) {
	client = new(Client)
	jsonstream, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(jsonstream, client); err != nil {
		return nil, err
	}
	client.Id = uuid.New()
	client.RegTime = time.Now()
	if client.Apps == nil {
		client.Apps = []string{}
	}
	client.SkipWorks = []string{}
	client.Status = "active"
	client.Tag = true
	return
}
