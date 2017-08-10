package core

import (
	"encoding/json"
	"fmt"
	"github.com/MG-RAST/AWE/lib/core/uuid"
	"github.com/MG-RAST/AWE/lib/logger"
	"io/ioutil"
	"time"
)

// states
//online bool // server defined
//suspended bool // server defined
//busy bool

// this is the Worker
type Client struct {
	coAckChannel    chan CoAck `bson:"-" json:"-"` //workunit checkout item including data and err (qmgr.Handler -> WorkController)
	RWMutex         `bson:"-" json:"-"`
	ClientReports   `bson:",inline" json:",inline"`
	RegTime         time.Time `bson:"regtime" json:"regtime"`
	LastCompleted   time.Time `bson:"lastcompleted" json:"lastcompleted"` // time of last time a job was completed (can be used to compute idle time)
	Serve_time      string    `bson:"serve_time" json:"serve_time"`
	Total_checkout  int       `bson:"total_checkout" json:"total_checkout"`
	Total_completed int       `bson:"total_completed" json:"total_completed"`
	Total_failed    int       `bson:"total_failed" json:"total_failed"`
	Skip_work       []string  `bson:"skip_work" json:"skip_work"`
	Last_failed     int       `bson:"-" json:"-"`
	Tag             bool      `bson:"-" json:"-"`
	Proxy           bool      `bson:"proxy" json:"proxy"`
	SubClients      int       `bson:"subclients" json:"subclients"`
	Online          bool      `bson:"online" json:"online"`
	Suspended       bool      `bson:"suspended" json:"suspended"`
	Status          string    `bson:"Status" json:"Status"` // 1) suspended? 2) busy ? 3) online (call is idle) 4) offline
	//Assigned_work      []string                            `bson:"assigned_work" json:"assigned_work"` // this is for exporting into json
	//_assigned_work_map map[Workunit_Unique_Identifier]bool `bson:"-" json:"-"`                         // this if for internal handling
	Assigned_work *WorkunitList `bson:"assigned_work" json:"assigned_work"` // this is for exporting into json
}

type ClientReports struct {
	Id            string        `bson:"id" json:"id"`     // this is a uuid (the only relevant identifier)
	Name          string        `bson:"name" json:"name"` // this can be anything you want
	Group         string        `bson:"group" json:"group"`
	User          string        `bson:"user" json:"user"`
	Domain        string        `bson:"domain" json:"domain"`
	Busy          bool          `bson:"busy" json:"busy"`
	InstanceId    string        `bson:"instance_id" json:"instance_id"`     // Openstack specific
	InstanceType  string        `bson:"instance_type" json:"instance_type"` // Openstack specific
	Host          string        `bson:"host" json:"host"`                   // deprecated
	Hostname      string        `bson:"hostname" json:"hostname"`
	Host_ip       string        `bson:"host_ip" json:"host_ip"` // Host can be physical machine or VM, whatever is helpful for management
	CPUs          int           `bson:"cores" json:"cores"`
	Apps          []string      `bson:"apps" json:"apps"`
	Current_work  *WorkunitList `bson:"current_work" json:"current_work"`
	GitCommitHash string        `bson:"git_commit_hash" json:"git_commit_hash"`
	Version       string        `bson:"version" json:"version"`
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
	if client.Assigned_work == nil {
		client.Assigned_work = NewWorkunitList()
	}

	if client.Assigned_work == nil {
		client.Current_work = NewWorkunitList()
	}

}

func NewClient() (client *Client) {
	client = &Client{
		Total_checkout:  0,
		Total_completed: 0,
		Total_failed:    0,

		Serve_time:    "0",
		Last_failed:   0,
		Status:        "offline",
		ClientReports: ClientReports{},
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

func (cl *Client) Append_Skip_work(workid Workunit_Unique_Identifier, write_lock bool) (err error) {
	if write_lock {
		err = cl.LockNamed("Append_Skip_work")
		if err != nil {
			return
		}
	}

	cl.Skip_work = append(cl.Skip_work, workid.String())
	if write_lock {
		cl.Unlock()
	}
	return
}

func (cl *Client) Contains_Skip_work_nolock(workid string) (c bool) {
	c = contains(cl.Skip_work, workid)
	return
}

func (cl *Client) Get_Id(do_read_lock bool) (s string, err error) {
	if do_read_lock {
		read_lock, xerr := cl.RLockNamed("Get_Id")
		if xerr != nil {
			err = xerr
			return
		}
		defer cl.RUnlockNamed(read_lock)
	}
	s = cl.Id
	return
}

// this function should not be used internally, this is only for backwards-compatibility and human readability
func (cl *Client) Get_New_Status(do_read_lock bool) (s string, err error) {
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

func (cl *Client) Get_Group(do_read_lock bool) (g string, err error) {
	if do_read_lock {
		read_lock, xerr := cl.RLockNamed("Get_Status")
		if xerr != nil {
			err = xerr
			return
		}
		defer cl.RUnlockNamed(read_lock)
	}
	g = cl.Group
	return
}

func (cl *Client) Set_Status_deprecated(s string, write_lock bool) (err error) {
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

func (cl *Client) Update_Status(write_lock bool) (err error) {
	if write_lock {
		err = cl.LockNamed("Update_Status")
		if err != nil {
			return
		}
		defer cl.Unlock()
	}

	// 1) suspended? 2) busy ? 3) online (call is idle) 4) offline

	if cl.Suspended {
		cl.Status = "suspended"
		return
	}

	if cl.Busy {
		cl.Status = "busy"
		return
	}

	if cl.Online {
		cl.Status = "online"
		return
	}

	cl.Status = "offline"
	return
}

func (cl *Client) Suspend(write_lock bool) (err error) {
	if write_lock {
		err = cl.LockNamed("Suspend")
		if err != nil {
			return
		}
		defer cl.Unlock()
	}

	if cl.Suspended != true {
		cl.Suspended = true
		cl.Update_Status(false)
	}
	return
}

func (cl *Client) Resume(write_lock bool) (err error) {
	if write_lock {
		err = cl.LockNamed("Resume")
		if err != nil {
			return
		}
		defer cl.Unlock()
	}

	if cl.Suspended != false {
		cl.Suspended = false
		cl.Update_Status(false)
	}
	return
}

func (cl *Client) Get_Suspended(do_read_lock bool) (s bool, err error) {
	if do_read_lock {
		read_lock, xerr := cl.RLockNamed("Get_Suspended")
		if xerr != nil {
			err = xerr
			return
		}
		defer cl.RUnlockNamed(read_lock)
	}
	s = cl.Suspended
	return
}
func (cl *Client) Get_Busy(do_read_lock bool) (b bool, err error) {
	if do_read_lock {
		read_lock, xerr := cl.RLockNamed("Get_Busy")
		if xerr != nil {
			err = xerr
			return
		}
		defer cl.RUnlockNamed(read_lock)
	}
	b = cl.Busy
	return
}

func (cl *Client) Set_Busy(b bool, do_write_lock bool) (err error) {
	if do_write_lock {
		err = cl.LockNamed("Set_Busy")
		if err != nil {
			return
		}
		defer cl.Unlock()
	}
	cl.Busy = b

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
	cl.LastCompleted = time.Now()
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

func (cl *Client) Increment_last_failed(write_lock bool) (value int, err error) {
	if write_lock {
		err = cl.LockNamed("Increment_last_failed")
		if err != nil {
			return
		}
		defer cl.Unlock()
	}
	cl.Last_failed += 1
	value = cl.Last_failed
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

//func (cl *Client) Set_current_work(current_work_ids []string, do_write_lock bool) (err error) {
//	current_work_ids = []string{}
//	if do_write_lock {
//		err = cl.LockNamed("Set_current_work")
//		if err != nil {
//			return
//		}
//		defer cl.Unlock()
//	}
//
//	cl.Assigned_work = make(map[string]bool)
//	for _, workid := range current_work_ids {
//		cl.Assigned_work[workid] = true
//	}
//
//	return
//}

// TODO: Wolfgang: Can we use delete instead ?
//func (cl *Client) Assigned_work_false(workid string) (err error) {
//	err = cl.LockNamed("Assigned_work_false")
//	if err != nil {
//		return
//	}
//	defer cl.Unlock()
//	cl.Assigned_work[workid] = false
//	return
//}

func (cl *Client) Marshal() (result []byte, err error) {
	read_lock, err := cl.RLockNamed("Marshal")
	if err != nil {
		return
	}
	defer cl.RUnlockNamed(read_lock)
	result, err = json.Marshal(cl)
	return
}
