package core

import ()

var (
	QMgr    ResourceMgr
	Service string
)

type ClientMgr interface {
	RegisterNewClient(FormFiles) (*Client, error)
	ClientHeartBeat(string) (HBmsg, error)
	GetClient(string) (*Client, error)
	GetAllClients() []*Client
	DeleteClient(string)
	ClientChecker()
}

type WorkMgr interface {
	GetWorkById(string) (*Workunit, error)
	ShowWorkunits(string) []*Workunit
	CheckoutWorkunits(string, string, int) ([]*Workunit, error)
	NotifyWorkStatus(Notice)
}

type JobMgr interface {
	JobRegister() (string, error)
	EnqueueTasksByJobId(string, []*Task) error
	GetActiveJobs() map[string]*JobPerf
	GetSuspendJobs() map[string]bool
	SuspendJob(string, string) error
	ResumeSuspendedJob(string) error
	ResubmitJob(string) error
	DeleteJob(string) error
	DeleteSuspendedJobs() int
	InitMaxJid() error
	RecoverJobs() error
	FinalizeWorkPerf(string, string) error
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

func InitResMgr(service string) {
	if service == "server" {
		QMgr = NewServerMgr()
	} else if service == "proxy" {
		QMgr = NewProxyMgr()
	}
	Service = service
}
