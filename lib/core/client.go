package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/MG-RAST/AWE/lib/core/uuid"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/rwmutex"
)

// states
//online bool // server defined
//suspended bool // server defined
//busy bool

// Client this is the Worker
type Client struct {
	coAckChannel    chan CoAck //workunit checkout item including data and err (qmgr.Handler -> WorkController)
	rwmutex.RWMutex `bson:"-" json:"-"`
	WorkerRuntime   `bson:",inline" json:",inline"`
	WorkerState     `bson:",inline" json:",inline"`
	RegTime         time.Time     `bson:"regtime" json:"regtime"`
	LastCompleted   time.Time     `bson:"lastcompleted" json:"lastcompleted"` // time of last time a job was completed (can be used to compute idle time)
	ServeTime       string        `bson:"serve_time" json:"serve_time"`
	TotalCheckout   int           `bson:"total_checkout" json:"total_checkout"`
	TotalCompleted  int           `bson:"total_completed" json:"total_completed"`
	TotalFailed     int           `bson:"total_failed" json:"total_failed"`
	SkipWork        []string      `bson:"skip_work" json:"skip_work"`
	LastFailed      int           `bson:"-" json:"-"`
	Tag             bool          `bson:"-" json:"-"`
	Proxy           bool          `bson:"proxy" json:"proxy"`
	SubClients      int           `bson:"subclients" json:"subclients"`
	Online          bool          `bson:"online" json:"online"`                 // a state
	Suspended       bool          `bson:"suspended" json:"suspended"`           // a state
	SuspendReason   string        `bson:"suspend_reason" json:"suspend_reason"` // a state
	Status          string        `bson:"Status" json:"Status"`                 // 0) unhealthy 1) suspended? 2) busy ? 3) online (call is idle) 4) offline
	AssignedWork    *WorkunitList `bson:"assigned_work" json:"assigned_work"`   // this is for exporting into json
}

// WorkerRuntime worker info that does not change at runtime
type WorkerRuntime struct {
	ID           string `bson:"id" json:"id"`     // this is a uuid (the only relevant identifier)
	Name         string `bson:"name" json:"name"` // this can be anything you want
	Group        string `bson:"group" json:"group"`
	User         string `bson:"user" json:"user"`
	Domain       string `bson:"domain" json:"domain"`
	InstanceID   string `bson:"instance_id" json:"instance_id"`     // Openstack specific
	InstanceType string `bson:"instance_type" json:"instance_type"` // Openstack specific
	//Host          string   `bson:"host" json:"host"`                   // deprecated
	Hostname      string   `bson:"hostname" json:"hostname"`
	HostIP        string   `bson:"host_ip" json:"host_ip"` // Host can be physical machine or VM, whatever is helpful for management
	CPUs          int      `bson:"cores" json:"cores"`
	Apps          []string `bson:"apps" json:"apps"`
	GitCommitHash string   `bson:"git_commit_hash" json:"git_commit_hash"`
	Version       string   `bson:"version" json:"version"`
}

// WorkerState changes at runtime
type WorkerState struct {
	Healthy      bool          `bson:"healthy" json:"healthy"`
	ErrorMessage string        `bson:"error_message" json:"error_message"`
	Busy         bool          `bson:"busy" json:"busy"` // a state
	CurrentWork  *WorkunitList `bson:"current_work" json:"current_work"`
}

// NewWorkerState creates WorkerState
func NewWorkerState() (ws *WorkerState) {
	ws = &WorkerState{}
	ws.Healthy = true
	ws.CurrentWork = NewWorkunitList()
	return
}

// Init : invoked by NewClient or manually after unmarshalling
func (client *Client) Init() {
	client.coAckChannel = make(chan CoAck)

	client.Tag = true

	if client.ID == "" {
		client.ID = uuid.New()
	}
	client.RWMutex.Init("client_" + client.ID)

	if client.RegTime.IsZero() {
		client.RegTime = time.Now()
	}
	if client.Apps == nil {
		client.Apps = []string{}
	}
	if client.SkipWork == nil {
		client.SkipWork = []string{}
	}

	client.AssignedWork.Init("Assigned_work")

	client.CurrentWork.Init("Current_work")

}

// NewClient _
func NewClient() (client *Client) {
	client = &Client{
		TotalCheckout:  0,
		TotalCompleted: 0,
		TotalFailed:    0,
		ServeTime:      "0",
		LastFailed:     0,
		Suspended:      false,
		//Status:          "online",
		Online:       true,
		AssignedWork: NewWorkunitList(),

		WorkerRuntime: WorkerRuntime{},
		WorkerState:   *NewWorkerState(),
	}
	client.Update_Status(false)

	client.Init()

	return
}

// NewProfileClient : create Client object from json file
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
		client = nil
		return
	}

	client.Init()

	return
}

// Add _
func (client *Client) Add(workid Workunit_Unique_Identifier) (err error) {
	err = client.AssignedWork.Add(workid)
	if err != nil {
		return
	}

	client.TotalCheckout++ // TODO add lock ????
	return
}

// GetAck _
func (client *Client) GetAck() (ack CoAck, err error) {
	startTime := time.Now()
	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(60 * time.Second)
		timeout <- true
	}()

	select {
	case ack = <-client.coAckChannel:
		elapsedTime := time.Since(startTime)
		logger.Debug(3, "got ack after %s", elapsedTime)
	case <-timeout:
		elapsedTime := time.Since(startTime)
		err = fmt.Errorf("(CheckoutWorkunits) %s workunit request timed out after %s ", client.ID, elapsedTime)
		return
	}

	return
}

// AppendSkipwork _
func (client *Client) AppendSkipwork(workid Workunit_Unique_Identifier, writeLock bool) (err error) {
	if writeLock {
		err = client.LockNamed("Append_Skip_work")
		if err != nil {
			return
		}
		defer client.Unlock()
	}

	var work_str string
	work_str, err = workid.String()
	if err != nil {
		err = fmt.Errorf("() workid.String() returned: %s", err.Error())
		return
	}

	client.SkipWork = append(client.SkipWork, work_str)

	return
}

func (client *Client) Contains_Skip_work_nolock(workid string) (c bool) {
	c = contains(client.SkipWork, workid)
	return
}

func (client *Client) Get_Id(do_read_lock bool) (s string, err error) {
	if do_read_lock {
		read_lock, xerr := client.RLockNamed("Get_Id")
		if xerr != nil {
			err = xerr
			return
		}
		defer client.RUnlockNamed(read_lock)
	}
	s = client.ID
	return
}

// this function should not be used internally, this is only for backwards-compatibility and human readability
func (client *Client) Get_New_Status(do_read_lock bool) (s string, err error) {
	if do_read_lock {
		read_lock, xerr := client.RLockNamed("Get_New_Status")
		if xerr != nil {
			err = xerr
			return
		}
		defer client.RUnlockNamed(read_lock)
	}
	s = client.Status
	return
}

func (client *Client) Get_Group(do_read_lock bool) (g string, err error) {
	if do_read_lock {
		read_lock, xerr := client.RLockNamed("Get_Group")
		if xerr != nil {
			err = xerr
			return
		}
		defer client.RUnlockNamed(read_lock)
	}
	g = client.Group
	return
}

func (client *Client) Set_Status_deprecated(s string, writeLock bool) (err error) {
	if writeLock {
		err = client.LockNamed("Set_Status")
		if err != nil {
			return
		}
		defer client.Unlock()
	}
	client.Status = s

	return
}

func (client *Client) Update_Status(writeLock bool) (err error) {
	if writeLock {
		err = client.LockNamed("Update_Status")
		if err != nil {
			return
		}
		defer client.Unlock()
	}

	// 1) suspended? 2) busy ? 3) online (call is idle) 4) offline

	if !client.Healthy {
		client.Status = "unhealthy"
		return
	}

	if client.Suspended {
		client.Status = "suspended"
		return
	}

	if client.Busy {
		client.Status = "busy"
		return
	}

	if client.Online {
		client.Status = "online"
		return
	}

	client.Status = "offline"
	return
}

func (client *Client) Set_Suspended(s bool, reason string, writeLock bool) (err error) {
	if writeLock {
		err = client.LockNamed("Set_Suspended")
		if err != nil {
			return
		}
		defer client.Unlock()
	}

	if client.Suspended != s {
		client.Suspended = s
		if s {
			if reason == "" {
				panic("suspending without providing eason not allowed !")
			}
		}
		client.SuspendReason = reason
		client.Update_Status(false)
	}
	return
}

func (client *Client) Suspend(reason string, writeLock bool) (err error) {
	return client.Set_Suspended(true, reason, writeLock)
}

func (client *Client) Resume(writeLock bool) (err error) {
	return client.Set_Suspended(false, "", writeLock)
}

func (client *Client) Get_Suspended(do_read_lock bool) (s bool, err error) {
	if do_read_lock {
		read_lock, xerr := client.RLockNamed("Get_Suspended")
		if xerr != nil {
			err = xerr
			return
		}
		defer client.RUnlockNamed(read_lock)
	}
	s = client.Suspended
	return
}

func (client *Client) Set_Online(o bool, writeLock bool) (err error) {
	if writeLock {
		err = client.LockNamed("Set_Online")
		if err != nil {
			return
		}
		defer client.Unlock()
	}

	if client.Online != o {
		client.Online = o
		client.Update_Status(false)
	}
	return
}

func (client *Client) Set_Busy(b bool, do_writeLock bool) (err error) {
	if do_writeLock {
		err = client.LockNamed("Set_Busy")
		if err != nil {
			return
		}
		defer client.Unlock()
	}
	if client.Busy != b {
		client.Busy = b
		client.Update_Status(false)
	}
	return
}

func (client *Client) Get_Busy(do_read_lock bool) (b bool, err error) {
	if do_read_lock {
		read_lock, xerr := client.RLockNamed("Get_Busy")
		if xerr != nil {
			err = xerr
			return
		}
		defer client.RUnlockNamed(read_lock)
	}
	b = client.Busy
	return
}

// GetTotalCheckout _
func (client *Client) GetTotalCheckout() (count int, err error) {
	readLock, err := client.RLockNamed("GetTotalCheckout")
	if err != nil {
		return
	}
	defer client.RUnlockNamed(readLock)
	count = client.TotalCheckout

	return
}

// IncrementTotalCheckout _
func (client *Client) IncrementTotalCheckout(err error) {
	err = client.LockNamed("IncrementTotalCheckout")
	if err != nil {
		return
	}
	defer client.Unlock()
	client.TotalCheckout += 1
	return
}

func (client *Client) Get_Total_completed() (count int, err error) {
	read_lock, err := client.RLockNamed("Get_Total_completed")
	if err != nil {
		return
	}
	defer client.RUnlockNamed(read_lock)
	count = client.TotalCompleted

	return
}

func (client *Client) Increment_total_completed() (err error) {
	err = client.LockNamed("Increment_total_completed")
	if err != nil {
		return
	}
	defer client.Unlock()
	client.LastCompleted = time.Now()
	client.TotalCompleted += 1
	client.LastFailed = 0 //reset last consecutive failures
	return
}

func (client *Client) Get_Total_failed() (count int, err error) {
	read_lock, err := client.RLockNamed("Get_Total_failed")
	if err != nil {
		return
	}
	defer client.RUnlockNamed(read_lock)
	count = client.TotalFailed

	return
}

func (client *Client) Increment_total_failed(writeLock bool) (err error) {
	if writeLock {
		err = client.LockNamed("Increment_total_failed")
		if err != nil {
			return
		}
		defer client.Unlock()
	}
	client.TotalFailed += 1

	return
}

func (client *Client) Increment_last_failed(writeLock bool) (value int, err error) {
	if writeLock {
		err = client.LockNamed("Increment_last_failed")
		if err != nil {
			return
		}
		defer client.Unlock()
	}
	client.LastFailed += 1
	value = client.LastFailed
	return
}

func (client *Client) Get_Last_failed() (count int, err error) {
	read_lock, err := client.RLockNamed("Get_Last_failed")
	if err != nil {
		return
	}
	defer client.RUnlockNamed(read_lock)
	count = client.LastFailed

	return
}

//func (client *Client) Set_current_work(current_work_ids []string, do_writeLock bool) (err error) {
//	current_work_ids = []string{}
//	if do_writeLock {
//		err = client.LockNamed("Set_current_work")
//		if err != nil {
//			return
//		}
//		defer client.Unlock()
//	}
//
//	client.Assigned_work = make(map[string]bool)
//	for _, workid := range current_work_ids {
//		client.Assigned_work[workid] = true
//	}
//
//	return
//}

// TODO: Wolfgang: Can we use delete instead ?
//func (client *Client) Assigned_work_false(workid string) (err error) {
//	err = client.LockNamed("Assigned_work_false")
//	if err != nil {
//		return
//	}
//	defer client.Unlock()
//	client.Assigned_work[workid] = false
//	return
//}

func (client *Client) Marshal() (result []byte, err error) {
	read_lock, err := client.RLockNamed("Marshal")
	if err != nil {
		return
	}
	defer client.RUnlockNamed(read_lock)
	result, err = json.Marshal(client)
	return
}
