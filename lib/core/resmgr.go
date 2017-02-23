package core

import (
	"github.com/MG-RAST/AWE/lib/user"
)

type ClientMgr interface {
	RegisterNewClient(FormFiles, *ClientGroup) (*Client, error)
	ClientHeartBeat(string, *ClientGroup) (HBmsg, error)
	GetClient(string, bool) (*Client, bool, error)
	GetClientByUser(string, *user.User) (*Client, error)
	//GetAllClients() []*Client
	GetClientMap() *ClientMap
	GetAllClientsByUser(*user.User) ([]*Client, error)
	DeleteClient(*Client) error
	DeleteClientById(string) error
	DeleteClientByUser(string, *user.User) error
	SuspendClient(string, *Client, bool) error
	SuspendClientByUser(string, *user.User) error
	ResumeClient(string) error
	ResumeClientByUser(string, *user.User) error
	ResumeSuspendedClients() (int, error)
	ResumeSuspendedClientsByUser(*user.User) int
	SuspendAllClients() (int, error)
	SuspendAllClientsByUser(*user.User) int
	ClientChecker()
	UpdateSubClients(string, int) error
	UpdateSubClientsByUser(string, int, *user.User)
}

type WorkMgr interface {
	GetWorkById(string) (*Workunit, error)
	ShowWorkunits(string) ([]*Workunit, error)
	ShowWorkunitsByUser(string, *user.User) []*Workunit
	CheckoutWorkunits(string, string, *Client, int64, int) ([]*Workunit, error)
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
	UpdateQueueJobInfo(*Job) error
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
	UpdateQueueLoop()
	NoticeHandle()
	GetJsonStatus() (map[string]map[string]int, error)
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
