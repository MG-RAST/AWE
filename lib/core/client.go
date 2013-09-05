package core

import (
	"encoding/json"
	"github.com/MG-RAST/AWE/lib/core/uuid"
	"io/ioutil"
	"time"
)

const (
	CLIENT_STAT_ACTIVE  = "active"
	CLIENT_STAT_SUSPEND = "suspend"
)

type Client struct {
	Id              string          `bson:"id" json:"id"`
	Name            string          `bson:"name" json:"name"`
	Group           string          `bson:"group" json:"group"`
	User            string          `bson:"user" json:"user"`
	Host            string          `bson:"host" json:"host"`
	CPUs            int             `bson:"cores" json:"cores"`
	Apps            []string        `bson:"apps" json:"apps"`
	RegTime         time.Time       `bson:"regtime" json:"regtime"`
	Serve_time      string          `bson:"serve_time" json:"serve_time"`
	Idle_time       int             `bson:"idle_time" json:"idle_time"`
	Status          string          `bson:"Status" json:"Status"`
	Total_checkout  int             `bson:"total_checkout" json:"total_checkout"`
	Total_completed int             `bson:"total_completed" json:"total_completed"`
	Total_failed    int             `bson:"total_failed" json:"total_failed"`
	Current_work    map[string]bool `bson:"current_work" json:"current_work"`
	Skip_work       []string        `bson:"skip_work" json:"skip_work"`
	Last_failed     int             `bson:"-" json:"-"`
	Tag             bool            `bson:"-" json:"-"`
	Proxy           bool            `bson:"proxy" json:"proxy"`
	SubClients      int             `bson:"subclients" json:"subclients"`
}

func NewClient() (client *Client) {
	client = new(Client)
	client.Id = uuid.New()
	client.Apps = []string{}
	client.Skip_work = []string{}
	client.Status = CLIENT_STAT_ACTIVE
	client.Total_checkout = 0
	client.Total_completed = 0
	client.Total_failed = 0
	client.Current_work = map[string]bool{}
	client.Tag = true
	client.Serve_time = "0"
	client.Last_failed = 0
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
	if client.Id == "" {
		client.Id = uuid.New()
	}
	if client.RegTime.IsZero() {
		client.RegTime = time.Now()
	}
	if client.Apps == nil {
		client.Apps = []string{}
	}
	client.Skip_work = []string{}
	client.Status = "active"
	if client.Current_work == nil {
		client.Current_work = map[string]bool{}
	}
	client.Tag = true
	return
}

func (cl *Client) IsBusy() bool {
	if len(cl.Current_work) > 0 {
		return true
	}
	return false
}
