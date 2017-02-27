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
	Current_work    map[string]bool `bson:"current_work" json:"current_work"` // the bool in the mapping is deprecated. It used to indicate completed work that could not be returned to server
	Skip_work       []string        `bson:"skip_work" json:"skip_work"`
	Last_failed     int             `bson:"-" json:"-"`
	Tag             bool            `bson:"-" json:"-"`
	Proxy           bool            `bson:"proxy" json:"proxy"`
	SubClients      int             `bson:"subclients" json:"subclients"`
	GitCommitHash   string          `bson:"git_commit_hash" json:"git_commit_hash"`
	Version         string          `bson:"version" json:"version"`
}

// invoked by NewClient or manually after unmarshalling
func (client *Client) Init() {
	client.coAckChannel = make(chan CoAck)

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
		Status:      CLIENT_STAT_ACTIVE_IDLE,
	}

	client.Init()

	return
}

// create Client object from json file
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

func (cl *Client) Append_Skip_work(workid string, write_lock bool) (err error) {
	if write_lock {
		err = cl.LockNamed("Append_Skip_work")
		if err != nil {
			return
		}
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

func (cl *Client) Get_Status(do_read_lock bool) (s string, err error) {
	if do_read_lock {
		read_lock, xerr := cl.RLockNamed("Get_Status")
		if xerr != nil {
			err = xerr
			return
		}
		defer cl.RUnlockNamed(read_lock)
	}
	s = cl.Status
	return
}

func (cl *Client) Set_Status(s string, write_lock bool) (err error) {
	if write_lock {
		err = cl.LockNamed("Set_Status")
		if err != nil {
			return
		}
		defer cl.Unlock()
	}
	cl.Status = s

	return
}

func (cl *Client) Get_Total_checkout() (count int, err error) {
	read_lock, err := cl.RLockNamed("Get_Total_checkout")
	if err != nil {
		return
	}
	defer cl.RUnlockNamed(read_lock)
	count = cl.Total_checkout

	return
}

func (cl *Client) Increment_total_checkout(err error) {
	err = cl.LockNamed("Increment_total_checkout")
	if err != nil {
		return
	}
	defer cl.Unlock()
	cl.Total_checkout += 1
	return
}

func (cl *Client) Get_Total_completed() (count int, err error) {
	read_lock, err := cl.RLockNamed("Get_Total_completed")
	if err != nil {
		return
	}
	defer cl.RUnlockNamed(read_lock)
	count = cl.Total_completed

	return
}

func (cl *Client) Increment_total_completed() (err error) {
	err = cl.LockNamed("Increment_total_completed")
	if err != nil {
		return
	}
	defer cl.Unlock()
	cl.Total_completed += 1
	cl.Last_failed = 0 //reset last consecutive failures
	return
}

func (cl *Client) Get_Total_failed() (count int, err error) {
	read_lock, err := cl.RLockNamed("Get_Total_failed")
	if err != nil {
		return
	}
	defer cl.RUnlockNamed(read_lock)
	count = cl.Total_failed

	return
}

func (cl *Client) Increment_total_failed(write_lock bool) (err error) {
	if write_lock {
		err = cl.LockNamed("Increment_total_failed")
		if err != nil {
			return
		}
		defer cl.Unlock()
	}
	cl.Total_failed += 1

	return
}

func (cl *Client) Get_Last_failed() (count int, err error) {
	read_lock, err := cl.RLockNamed("Get_Last_failed")
	if err != nil {
		return
	}
	defer cl.RUnlockNamed(read_lock)
	count = cl.Last_failed

	return
}

func (cl *Client) Increment_last_failed(err error) {
	err = cl.LockNamed("Increment_last_failed")
	defer cl.Unlock()
	cl.Last_failed += 1

	return
}

func (cl *Client) Current_work_delete(workid string, write_lock bool) (err error) {
	if write_lock {
		err = cl.LockNamed("Current_work_delete")
		defer cl.Unlock()
	}
	delete(cl.Current_work, workid)
	cw_length, err := cl.Current_work_length(false)
	if err != nil {
		return
	}

	if cw_length == 0 && cl.Status == CLIENT_STAT_ACTIVE_BUSY {
		cl.Status = CLIENT_STAT_ACTIVE_IDLE
	}
	return
}

func (cl *Client) Get_current_work(do_read_lock bool) (current_work_ids []string, err error) {
	current_work_ids = []string{}
	if do_read_lock {
		read_lock, xerr := cl.RLockNamed("Get_current_work")
		if xerr != nil {
			err = xerr
			return
		}
		defer cl.RUnlockNamed(read_lock)
	}
	for id := range cl.Current_work {
		current_work_ids = append(current_work_ids, id)
	}
	return
}

// TODO: Wolfgang: Can we use delete instead ?
//func (cl *Client) Current_work_false(workid string) (err error) {
//	err = cl.LockNamed("Current_work_false")
//	if err != nil {
//		return
//	}
//	defer cl.Unlock()
//	cl.Current_work[workid] = false
//	return
//}

// lock always
func (cl *Client) Add_work(workid string) (err error) {

	err = cl.LockNamed("Add_work")
	if err != nil {
		return
	}
	defer cl.Unlock()

	cl.Current_work[workid] = true
	cl.Total_checkout += 1
	return
}

func (cl *Client) Current_work_length(lock bool) (clength int, err error) {
	if lock {
		read_lock, xerr := cl.RLockNamed("Current_work_length")
		if xerr != nil {
			err = xerr
			return
		}
		defer cl.RUnlockNamed(read_lock)
	}
	clength = len(cl.Current_work)

	return
}

func (cl *Client) IsBusy(lock bool) (busy bool, err error) {
	cw_length, err := cl.Current_work_length(lock)
	if err != nil {
		return
	}
	if cw_length > 0 {
		busy = true
		return
	}
	busy = false
	return
}

func (cl *Client) Marshal() (result []byte, err error) {
	read_lock, err := cl.RLockNamed("Marshal")
	if err != nil {
		return
	}
	defer cl.RUnlockNamed(read_lock)
	result, err = json.Marshal(cl)
	return
}
