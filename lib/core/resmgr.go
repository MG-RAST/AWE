package core

import (
	"github.com/MG-RAST/AWE/lib/user"
)

type ClientMgr interface {
	RegisterNewClient(FormFiles, *ClientGroup) (*Client, error)
	ClientHeartBeat(string, *ClientGroup) (HBmsg, error)
	GetClient(string) (*Client, bool)
	GetClientByUser(string, *user.User) (*Client, error)
	GetAllClients() []*Client
	GetAllClientsByUser(*user.User) []*Client
	DeleteClient(string) error
	DeleteClientByUser(string, *user.User) error
	SuspendClient(string) error
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
	JobRegister() (string, error)
	EnqueueTasksByJobId(string, []*Task) error
	GetActiveJobs() map[string]bool
	IsJobRegistered(string) bool
	GetSuspendJobs() map[string]bool
	SuspendJob(string, string, string) error
	ResumeSuspendedJob(string) error
	ResumeSuspendedJobByUser(string, *user.User) error
	ResumeSuspendedJobs() int
	ResumeSuspendedJobsByUser(*user.User) int
	ResubmitJob(string) error
	DeleteJob(string) error
	DeleteJobByUser(string, *user.User) error
	DeleteSuspendedJobs() int
	DeleteSuspendedJobsByUser(*user.User) int
	DeleteZombieJobs() int
	DeleteZombieJobsByUser(*user.User) int
	InitMaxJid() error
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
	JidHandle()
	TaskHandle()
	ClientHandle()
	ShowStatus() string
	QueueStatus() string
	SuspendQueue()
	ResumeQueue()
}
