package core

import ()

type ClientMgr interface {
	RegisterNewClient(FormFiles) (*Client, error)
	ClientHeartBeat(string) (HBmsg, error)
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
	ResumeSuspendedJobs() int
	ResubmitJob(string) error
	DeleteJob(string) error
	DeleteSuspendedJobs() int
	DeleteZombieJobs() int
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
