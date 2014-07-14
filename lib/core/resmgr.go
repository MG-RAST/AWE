package core

import (
	"github.com/MG-RAST/AWE/lib/user"
)

type ClientMgr interface {
	RegisterNewClient(FormFiles, *ClientGroup) (*Client, error)
	ClientHeartBeat(string, *ClientGroup) (HBmsg, error)
	GetClient(string) (*Client, error)
	GetAllClients() []*Client
	DeleteClient(string) (err error)
	SuspendClient(string) (err error)
	ResumeClient(string) (err error)
	ResumeSuspendedClients() (count int)
	SuspendAllClients() (count int)
	ClientChecker()
	UpdateSubClients(id string, count int)
}

type WorkMgr interface {
	GetWorkById(string) (*Workunit, error)
	ShowWorkunits(string) []*Workunit
	CheckoutWorkunits(string, string, int) ([]*Workunit, error)
	NotifyWorkStatus(Notice)
	EnqueueWorkunit(*Workunit) error
	FetchDataToken(string, string) (string, error)
	FetchPrivateEnv(string, string) (map[string]string, error)
}

type JobMgr interface {
	JobRegister() (string, error)
	EnqueueTasksByJobId(string, []*Task) error
	GetActiveJobs() map[string]*JobPerf
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
	Handle()
	ShowStatus() string
	Timer()
}
