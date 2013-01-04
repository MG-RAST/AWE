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
	User       string    `bson:"user" json:"user"`
	Host       string    `bson:"tasks" json:"tasks"`
	Workers    int       `bson:"workers" json:"workers"`
	LocalShock string    `bson:"localshock" json:localshock"`
	Cores      int       `bson:"cores" json:"cores"`
	Apps       []string  `bson:"apps" json:"apps"`
	SkipWorks  []string  `bson:"SkipWorkunits" json:"SkipWorkunits"`
	RegTime    time.Time `bson:"regtime" json:"regtime"`
}

func NewClient() (client *Client) {
	client = new(Client)
	client.Id = uuid.New()
	client.Apps = []string{}
	client.SkipWorks = []string{}
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
	return
}
