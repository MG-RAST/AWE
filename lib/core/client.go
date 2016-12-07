package core

import (
	"encoding/json"
	"fmt"
	"github.com/MG-RAST/AWE/lib/core/uuid"
	"github.com/MG-RAST/AWE/lib/logger"
	"io/ioutil"
	"time"
)

const (
	CLIENT_STAT_ACTIVE_BUSY = "active-busy"
	CLIENT_STAT_ACTIVE_IDLE = "active-idle"
	CLIENT_STAT_SUSPEND     = "suspend"
	CLIENT_STAT_DELETED     = "deleted"
)

type Client struct {
	coAckChannel chan CoAck `bson:"-" json:"-"` //workunit checkout item including data and err (qmgr.Handler -> WorkController)
	RWMutex
	Id              string          `bson:"id" json:"id"`
	Name            string          `bson:"name" json:"name"`
	Group           string          `bson:"group" json:"group"`
	User            string          `bson:"user" json:"user"`
	Domain          string          `bson:"domain" json:"domain"`
	InstanceId      string          `bson:"instance_id" json:"instance_id"`
	InstanceType    string          `bson:"instance_type" json:"instance_type"`
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
	GitCommitHash   string          `bson:"git_commit_hash" json:"git_commit_hash"`
	Version         string          `bson:"version" json:"version"`
}

func NewClient() (client *Client) {
	client = new(Client)
	client.coAckChannel = make(chan CoAck)
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
	client.RWMutex.Name = "client"
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

	client.coAckChannel = make(chan CoAck)

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
	client.RWMutex.Name = "client"
	return
}

func (cl *Client) Get_Ack() (ack CoAck, err error) {
	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(10 * time.Second) // TODO set this higher
		timeout <- true
	}()

	select {
	case ack = <-cl.coAckChannel:
		logger.Debug(3, "got ack")
	case <-timeout:
		err = fmt.Errorf("(CheckoutWorkunits) workunit request timed out")
		return
	}

	return
}

func (cl *Client) Append_Skip_work(workid string, write_lock bool) {
	if write_lock {
		cl.LockNamed("Append_Skip_work")
	}
	cl.Skip_work = append(cl.Skip_work, workid)
	if write_lock {
		cl.Unlock()
	}
	return
}

func (cl *Client) Contains_Skip_work_nolock(workid string) (c bool) {
	c = contains(cl.Skip_work, workid)
	return
}

func (cl *Client) Get_Status(read_lock bool) (s string) {
	if read_lock {
		cl.RLock()
	}
	s = cl.Status
	if read_lock {
		cl.RUnlock()
	}
	return
}

func (cl *Client) Set_Status(s string, write_lock bool) {
	if write_lock {
		cl.LockNamed("Set_Status")
		defer cl.Unlock()
	}
	cl.Status = s

	return
}

func (cl *Client) Get_Total_checkout() (count int) {
	cl.RLock()
	count = cl.Total_checkout
	cl.RUnlock()
	return
}

func (cl *Client) Increment_total_checkout() {
	cl.LockNamed("Increment_total_checkout")
	defer cl.Unlock()
	cl.Total_checkout += 1
	return
}

func (cl *Client) Get_Total_completed() (count int) {
	cl.RLock()
	count = cl.Total_completed
	cl.RUnlock()
	return
}

func (cl *Client) Increment_total_completed() {
	cl.LockNamed("Increment_total_completed")
	defer cl.Unlock()
	cl.Total_completed += 1

	return
}

func (cl *Client) Get_Total_failed() (count int) {
	cl.RLock()
	count = cl.Total_failed
	cl.RUnlock()
	return
}

func (cl *Client) Increment_total_failed(write_lock bool) {
	if write_lock {
		cl.LockNamed("Increment_total_failed")
		defer cl.Unlock()
	}
	cl.Total_failed += 1

	return
}

func (cl *Client) Get_Last_failed() (count int) {
	cl.RLock()
	count = cl.Last_failed
	cl.RUnlock()
	return
}

func (cl *Client) Increment_last_failed() {
	cl.LockNamed("Increment_last_failed")
	defer cl.Unlock()
	cl.Last_failed += 1

	return
}

func (cl *Client) Current_work_delete(workid string, write_lock bool) {
	if write_lock {
		cl.LockNamed("Current_work_delete")
		defer cl.Unlock()
	}
	delete(cl.Current_work, workid)

}

// TODO: Wolfgang: Can we use delete instead ?
func (cl *Client) Current_work_false(workid string) {
	cl.LockNamed("Current_work_false")
	defer cl.Unlock()
	cl.Current_work[workid] = false
}

// _nolock assumes you already have global lock
func (cl *Client) Add_work_nolock(workid string) {
	cl.Current_work[workid] = true
	cl.Total_checkout += 1
}

func (cl *Client) Add_work(workid string) {
	cl.LockNamed("Add_work")
	defer cl.Unlock()
	cl.Add_work_nolock(workid)
}

func (cl *Client) Current_work_length(lock bool) (clength int) {
	if lock {
		cl.RLock()
	}
	clength = len(cl.Current_work)
	if lock {
		cl.RUnlock()
	}
	return clength
}

func (cl *Client) IsBusy(lock bool) bool {
	if cl.Current_work_length(lock) > 0 {
		return true
	}
	return false
}
