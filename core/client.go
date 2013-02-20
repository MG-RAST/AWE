package core

import (
	"encoding/json"
	"github.com/MG-RAST/AWE/core/uuid"
	"io/ioutil"
	"time"
)

type Client struct {
	Id              string          `bson:"id" json:"id"`
	Name            string          `bson:"name" json:"name"`
	Group           string          `bson:"group" json:"group"`
	User            string          `bson:"user" json:"user"`
	Host            string          `bson:"host" json:"host"`
	Workers         int             `bson:"workers" json:"workers"`
	LocalShock      string          `bson:"localshock" json:"-"`
	Cores           int             `bson:"cores" json:"cores"`
	Apps            []string        `bson:"apps" json:"apps"`
	RegTime         time.Time       `bson:"regtime" json:"regtime"`
	Serve_time      string          `bson:"serve_time" json:"serve_time"`
	Status          string          `bson:"Status" json:"Status"`
	Total_checkout  int             `bson:"total_checkout" json:"total_checkout"`
	Total_completed int             `bson:"total_completed" json:"total_completed"`
	Total_failed    int             `bson:"total_failed" json:"total_failed"`
	Current_work    map[string]bool `bson:"current_work" json:"current_work"`
	Skip_work       []string        `bson:"skip_work" json:"skip_work"`
	Tag             bool            `bson:"-" json:"-"`
}

func NewClient() (client *Client) {
	client = new(Client)
	client.Id = uuid.New()
	client.Apps = []string{}
	client.Skip_work = []string{}
	client.Status = "active"
	client.Total_checkout = 0
	client.Total_completed = 0
	client.Total_failed = 0
	client.Current_work = map[string]bool{}
	client.Tag = true
	client.Serve_time = "0"
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
	client.Skip_work = []string{}
	client.Status = "active"
	client.Total_checkout = 0
	client.Total_completed = 0
	client.Total_failed = 0
	client.Current_work = map[string]bool{}
	client.Tag = true
	client.Serve_time = "0"
	return
}
