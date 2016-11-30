package core

import (
	"github.com/MG-RAST/AWE/lib/user"
	"sync"
)

type ClientMgr interface {
	RegisterNewClient(FormFiles, *ClientGroup) (*Client, error)
	ClientHeartBeat(string, *ClientGroup) (HBmsg, error)
	GetClient(string) (*Client, bool)
	GetClientByUser(string, *user.User) (*Client, error)
	//GetAllClients() []*Client
	GetClientMap() *ClientMap
	GetAllClientsByUser(*user.User) []*Client
	DeleteClient(*Client) error
	DeleteClientById(string) error
	DeleteClientByUser(string, *user.User) error
	SuspendClient(string, *Client, bool) error
	SuspendClientByUser(string, *user.User) error
	ResumeClient(string) error
	ResumeClientByUser(string, *user.User) error
	ResumeSuspendedClients() int
	ResumeSuspendedClientsByUser(*user.User) int
	SuspendAllClients() int
	SuspendAllClientsByUser(*user.User) int
	ClientChecker()
	UpdateSubClients(string, int)
	UpdateSubClientsByUser(string, int, *user.User)
}

type WorkMgr interface {
	GetWorkById(string) (*Workunit, error)
	ShowWorkunits(string) []*Workunit
	ShowWorkunitsByUser(string, *user.User) []*Workunit
	CheckoutWorkunits(string, string, int64, int) ([]*Workunit, error)
	NotifyWorkStatus(Notice)
	EnqueueWorkunit(*Workunit) error
	FetchDataToken(string, string) (string, error)
	FetchPrivateEnv(string, string) (map[string]string, error)
}

type JobMgr interface {
	EnqueueTasksByJobId(string, []*Task) error
	GetActiveJobs() map[string]bool
	IsJobRegistered(string) bool
	GetSuspendJobs() map[string]bool
	SuspendJob(string, string, string) error
	ResumeSuspendedJobByUser(string, *user.User) error
	ResumeSuspendedJobsByUser(*user.User) int
	ResubmitJob(string) error
	DeleteJobByUser(string, *user.User, bool) error
	DeleteSuspendedJobsByUser(*user.User, bool) int
	DeleteZombieJobsByUser(*user.User, bool) int
	RecoverJob(string) error
	RecoverJobs() error
	FinalizeWorkPerf(string, string) error
	SaveStdLog(string, string, string) error
	GetReportMsg(string, string) (string, error)
	RecomputeJob(string, string) error
	UpdateGroup(string, string) error
	UpdatePriority(string, int) error
}

type ClientWorkMgr interface {
	ClientMgr
	WorkMgr
}

type ResourceMgr interface {
	ClientWorkMgr
	JobMgr
	TaskHandle()
	ClientHandle()
	GetJsonStatus() map[string]map[string]int
	GetTextStatus() string
	QueueStatus() string
	GetQueue(string) interface{}
	SuspendQueue()
	ResumeQueue()
	Lock()
	Unlock()
	RLock()
	RUnlock()
}

type ClientMap struct {
	sync.RWMutex
	_map map[string]*Client
}

func NewClientMap() *ClientMap {
	return &ClientMap{_map: make(map[string]*Client)}
}

func (cl *ClientMap) GetMap() *map[string]*Client {
	return &cl._map
}

func (cl *ClientMap) Add(client *Client, lock bool) {
	if lock {
		cl.Lock()
	}
	cl._map[client.Id] = client
	if lock {
		cl.Unlock()
	}
}

func (cl *ClientMap) Get(client_id string) (client *Client, ok bool) {
	cl.RLock()
	client, ok = cl._map[client_id]
	cl.RUnlock()
	return
}

func (cl *ClientMap) Delete(client_id string, lock bool) {
	if lock {
		cl.Lock()
	}
	delete(cl._map, client_id)
	if lock {
		cl.Unlock()
	}
	return
}

func (cl *ClientMap) GetClientIds() (ids []string) {
	cl.RLock()
	for id, _ := range cl._map {
		ids = append(ids, id)
	}
	cl.RUnlock()
	return
}
