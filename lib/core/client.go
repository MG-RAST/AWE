package core

import (
	"encoding/json"
	"github.com/MG-RAST/AWE/lib/core/uuid"
	"github.com/MG-RAST/AWE/lib/logger"
	"io/ioutil"
	"sync"
	"time"
)

const (
	CLIENT_STAT_ACTIVE_BUSY = "active-busy"
	CLIENT_STAT_ACTIVE_IDLE = "active-idle"
	CLIENT_STAT_SUSPEND     = "suspend"
	CLIENT_STAT_DELETED     = "deleted"
)

type Client struct {
	Id                string          `bson:"id" json:"id"`
	Lock              sync.RWMutex    // Locks only members that can change. Current_work has its own lock.
	Name              string          `bson:"name" json:"name"`
	Group             string          `bson:"group" json:"group"`
	User              string          `bson:"user" json:"user"`
	Domain            string          `bson:"domain" json:"domain"`
	InstanceId        string          `bson:"instance_id" json:"instance_id"`
	InstanceType      string          `bson:"instance_type" json:"instance_type"`
	Host              string          `bson:"host" json:"host"`
	CPUs              int             `bson:"cores" json:"cores"`
	Apps              []string        `bson:"apps" json:"apps"`
	RegTime           time.Time       `bson:"regtime" json:"regtime"`
	Serve_time        string          `bson:"serve_time" json:"serve_time"`
	Idle_time         int             `bson:"idle_time" json:"idle_time"`
	Status            string          `bson:"Status" json:"Status"`
	Total_checkout    int             `bson:"total_checkout" json:"total_checkout"`
	Total_completed   int             `bson:"total_completed" json:"total_completed"`
	Total_failed      int             `bson:"total_failed" json:"total_failed"`
	Current_work      map[string]bool `bson:"current_work" json:"current_work"`
	Current_work_lock sync.RWMutex
	Skip_work         []string `bson:"skip_work" json:"skip_work"`
	Last_failed       int      `bson:"-" json:"-"`
	Tag               bool     `bson:"-" json:"-"`
	Proxy             bool     `bson:"proxy" json:"proxy"`
	SubClients        int      `bson:"subclients" json:"subclients"`
	GitCommitHash     string   `bson:"git_commit_hash" json:"git_commit_hash"`
	Version           string   `bson:"version" json:"version"`
}

func NewClient() (client *Client) {
	client = new(Client)
	client.Id = uuid.New()
	client.Apps = []string{}
	client.Skip_work = []string{}
	client.Status = CLIENT_STAT_ACTIVE_IDLE
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
		logger.Error("failed to unmashal json stream for client profile: " + string(jsonstream[:]))
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
	client.Status = CLIENT_STAT_ACTIVE_IDLE
	if client.Current_work == nil {
		client.Current_work = map[string]bool{}
	}
	client.Tag = true
	return
}

func (cl *Client) Append_Skip_work(workid string) {
	cl.Lock.Lock()
	cl.Skip_work = append(cl.Skip_work, workid)
	cl.Lock.Unlock()
	return
}

func (cl *Client) Contains_Skip_work(workid string) (c bool) {
	cl.Lock.RLock()
	c = contains(cl.Skip_work, workid)
	cl.Lock.RUnlock()
	return
}

func (cl *Client) Get_Status() (s string) {
	cl.Lock.RLock()
	s = cl.Status
	cl.Lock.RUnlock()
	return
}

func (cl *Client) Set_Status(s string) {
	cl.Lock.Lock()
	cl.Status = s
	cl.Lock.Unlock()
	return
}

func (cl *Client) Get_Total_checkout() (count int) {
	cl.Lock.RLock()
	count = cl.Total_checkout
	cl.Lock.RUnlock()
	return
}

func (cl *Client) Increment_total_checkout() {
	cl.Lock.Lock()
	cl.Total_checkout += 1
	cl.Lock.Unlock()
	return
}

func (cl *Client) Get_Total_completed() (count int) {
	cl.Lock.RLock()
	count = cl.Total_completed
	cl.Lock.RUnlock()
	return
}

func (cl *Client) Increment_total_completed() {
	cl.Lock.Lock()
	cl.Total_completed += 1
	cl.Lock.Unlock()
	return
}

func (cl *Client) Get_Total_failed() (count int) {
	cl.Lock.RLock()
	count = cl.Total_failed
	cl.Lock.RUnlock()
	return
}

func (cl *Client) Increment_total_failed() {
	cl.Lock.Lock()
	cl.Total_failed += 1
	cl.Lock.Unlock()
	return
}

func (cl *Client) Get_Last_failed() (count int) {
	cl.Lock.RLock()
	count = cl.Last_failed
	cl.Lock.RUnlock()
	return
}

func (cl *Client) Increment_last_failed() {
	cl.Lock.Lock()
	cl.Last_failed += 1
	cl.Lock.Unlock()
	return
}

func (cl *Client) Current_work_delete(workid string) {
	cl.Current_work_lock.Lock()
	delete(cl.Current_work, workid)
	cl.Current_work_lock.Unlock()
}

// TODO: Wolfgang: Can we use delete instead ?
func (cl *Client) Current_work_false(workid string) {
	cl.Current_work_lock.Lock()
	cl.Current_work[workid] = false
	cl.Current_work_lock.Unlock()
}

func (cl *Client) Current_work_add(workid string) {
	cl.Current_work_lock.Lock()
	cl.Current_work[workid] = true
	cl.Current_work_lock.Unlock()
}

func (cl *Client) Current_work_length() int {
	cl.Current_work_lock.RLock()
	clength := len(cl.Current_work)
	cl.Current_work_lock.RUnlock()
	return clength
}

func (cl *Client) IsBusy() bool {
	if cl.Current_work_length() > 0 {
		return true
	}
	return false
}
