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

func (client *Client) Init() {
	client.coAckChannel = make(chan CoAck)
	client.Status = CLIENT_STAT_ACTIVE_IDLE
	client.Tag = true

	if client.Id == "" {
		client.Id = uuid.New()
	}
	client.RWMutex.Init("client_" + client.Id)

	if client.RegTime.IsZero() {
		client.RegTime = time.Now()
	}
	if client.Apps == nil {
		client.Apps = []string{}
	}
	if client.Skip_work == nil {
		client.Skip_work = []string{}
	}
	if client.Current_work == nil {
		client.Current_work = map[string]bool{}
	}

}

func NewClient() (client *Client) {
	client = &Client{
		Total_checkout:  0,
		Total_completed: 0,
		Total_failed:    0,

		Serve_time:  "0",
		Last_failed: 0,
	}

	client.Init()

	return
}

func NewProfileClient(filepath string) (client *Client, err error) {

	jsonstream, err := ioutil.ReadFile(filepath)
	if err != nil {
		err = fmt.Errorf("(NewProfileClient) error in ioutil.ReadFile(filepath): %s %s", filepath, err.Error())
		return nil, err
	}

	if len(jsonstream) == 0 {
		err = fmt.Errorf("filepath %s seems to be empty", filepath)
		return
	}

	client = new(Client)
	err = json.Unmarshal(jsonstream, client)
	if err != nil {
		err = fmt.Errorf("failed to unmashal json stream for client profile (error: %s) (file: %s) json: %s", err.Error(), filepath, string(jsonstream[:]))
		return nil, err
	}

	client.Init()

	return
}

func (cl *Client) Get_Ack() (ack CoAck, err error) {
	start_time := time.Now()
	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(60 * time.Second)
		timeout <- true
	}()

	select {
	case ack = <-cl.coAckChannel:
		elapsed_time := time.Since(start_time)
		logger.Debug(3, "got ack after %s", elapsed_time)
	case <-timeout:
		elapsed_time := time.Since(start_time)
		err = fmt.Errorf("(CheckoutWorkunits) %s workunit request timed out after %s ", cl.Id, elapsed_time)
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
		read_lock := cl.RLockNamed("Get_Status")
		defer cl.RUnlockNamed(read_lock)
	}
	s = cl.Status
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
	read_lock := cl.RLockNamed("Get_Total_checkout")
	defer cl.RUnlockNamed(read_lock)
	count = cl.Total_checkout

	return
}

func (cl *Client) Increment_total_checkout() {
	cl.LockNamed("Increment_total_checkout")
	defer cl.Unlock()
	cl.Total_checkout += 1
	return
}

func (cl *Client) Get_Total_completed() (count int) {
	read_lock := cl.RLockNamed("Get_Total_completed")
	defer cl.RUnlockNamed(read_lock)
	count = cl.Total_completed

	return
}

func (cl *Client) Increment_total_completed() {
	cl.LockNamed("Increment_total_completed")
	defer cl.Unlock()
	cl.Total_completed += 1
	cl.Last_failed = 0 //reset last consecutive failures
	return
}

func (cl *Client) Get_Total_failed() (count int) {
	read_lock := cl.RLockNamed("Get_Total_failed")
	defer cl.RUnlockNamed(read_lock)
	count = cl.Total_failed

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
	read_lock := cl.RLockNamed("Get_Last_failed")
	defer cl.RUnlockNamed(read_lock)
	count = cl.Last_failed

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
	if cl.Current_work_length(false) == 0 && cl.Status == CLIENT_STAT_ACTIVE_BUSY {
		cl.Status = CLIENT_STAT_ACTIVE_IDLE
	}
}

func (cl *Client) Get_current_work(read_lock bool) (current_work_ids []string) {
	current_work_ids = []string{}
	if read_lock {
		read_lock := cl.RLockNamed("Get_current_work")
		defer cl.RUnlockNamed(read_lock)
	}
	for id := range cl.Current_work {
		current_work_ids = append(current_work_ids, id)
	}
	return
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
		read_lock := cl.RLockNamed("Current_work_length")
		defer cl.RUnlockNamed(read_lock)
	}
	clength = len(cl.Current_work)

	return clength
}

func (cl *Client) IsBusy(lock bool) bool {
	if cl.Current_work_length(lock) > 0 {
		return true
	}
	return false
}

func (cl *Client) Marshal() ([]byte, error) {
	read_lock := cl.RLockNamed("Marshal")
	defer cl.RUnlockNamed(read_lock)
	return json.Marshal(cl)
}
