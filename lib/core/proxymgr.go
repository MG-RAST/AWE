package core

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/AWE/lib/user"
	"os"
	"time"
)

type ProxyMgr struct {
	CQMgr
}

func NewProxyMgr() *ProxyMgr {
	return &ProxyMgr{
		CQMgr: CQMgr{
			clientMap: map[string]*Client{},
			workQueue: NewWQueue(),
			reminder:  make(chan bool),
			coReq:     make(chan CoReq),
			coAck:     make(chan CoAck),
			feedback:  make(chan Notice),
			coSem:     make(chan int, 1), //non-blocking buffered channel
		},
	}
}

//to-do: consider separate some independent tasks into another goroutine to handle

func (qm *ProxyMgr) Handle() {
	for {
		select {
		case coReq := <-qm.coReq:
			logger.Debug(2, fmt.Sprintf("proxymgr: workunit checkout request received, Req=%v\n", coReq))
			works, err := qm.popWorks(coReq)
			ack := CoAck{workunits: works, err: err}
			qm.coAck <- ack

		case notice := <-qm.feedback:
			logger.Debug(2, fmt.Sprintf("proxymgr: workunit feedback received, workid=%s, status=%s, clientid=%s\n", notice.WorkId, notice.Status, notice.ClientId))
			if err := qm.handleWorkStatusChange(notice); err != nil {
				logger.Error("handleWorkStatusChange(): " + err.Error())
			}
		case <-qm.reminder:
			logger.Debug(3, "time to update workunit queue....\n")
			if conf.DEV_MODE {
				fmt.Println(qm.ShowStatus())
			}
		}
	}
}

func (qm *ProxyMgr) Timer() {
	for {
		time.Sleep(10 * time.Second)
		qm.reminder <- true
	}
}

func (qm *ProxyMgr) InitMaxJid() (err error) {
	return
}

func (qm *ProxyMgr) ShowStatus() string {
	return ""
}

//---end of mgr methods

// workunit methods

//handle feedback from a client about the execution of a workunit
func (qm *ProxyMgr) handleWorkStatusChange(notice Notice) (err error) {
	//relay the notice to the server
	perf := new(WorkPerf)
	workid := notice.WorkId
	clientid := notice.ClientId
	if client, ok := qm.GetClient(clientid); ok {
		delete(client.Current_work, workid)
		if len(client.Current_work) == 0 {
			client.Status = CLIENT_STAT_ACTIVE_IDLE
		}
		qm.PutClient(client)
	}
	if work, ok := qm.workQueue.Get(workid); ok {
		work.State = notice.Status
		if err = proxy_relay_workunit(work, perf); err != nil {
			return
		}
		if work.State == WORK_STAT_DONE {
			if client, ok := qm.GetClient(clientid); ok {
				client.Total_completed += 1
				client.Last_failed = 0 //reset last consecutive failures
				qm.PutClient(client)
			}
		} else if work.State == WORK_STAT_FAIL {
			if client, ok := qm.GetClient(clientid); ok {
				client.Skip_work = append(client.Skip_work, workid)
				client.Total_failed += 1
				client.Last_failed += 1 //last consecutive failures
				if client.Last_failed == conf.MAX_CLIENT_FAILURE {
					client.Status = CLIENT_STAT_SUSPEND
				}
				qm.PutClient(client)
			}
		}
		qm.workQueue.Put(work)
	}
	return
}

func (qm *ProxyMgr) FetchDataToken(workid string, clientid string) (token string, err error) {
	return
}

func (qm *ProxyMgr) FetchPrivateEnv(workid string, clientid string) (env map[string]string, err error) {
	return
}

//end of workunits methods

//client methods

func (qm *ProxyMgr) RegisterNewClient(files FormFiles, cg *ClientGroup) (client *Client, err error) {
	if _, ok := files["profile"]; ok {
		client, err = NewProfileClient(files["profile"].Path)
		os.Remove(files["profile"].Path)
	} else {
		client = NewClient()
	}
	if err != nil {
		return nil, err
	}
	// If the name of the clientgroup does not match the name in the client profile, throw an error
	if cg != nil && client.Group != cg.Name {
		return nil, errors.New("Clientgroup name in token does not match that in the client configuration.")
	}
	qm.PutClient(client)
	if len(client.Current_work) > 0 { //re-registered client
		// move already checked-out workunit from waiting queue (workMap) to checked-out list (coWorkMap)
		for workid, _ := range client.Current_work {
			if qm.workQueue.Has(workid) {
				qm.workQueue.StatusChange(workid, WORK_STAT_CHECKOUT)
			}
		}
	}
	//proxy specific
	Self.SubClients += 1
	notifySubClients(Self.Id, Self.SubClients)
	return
}

func (qm *ProxyMgr) ClientChecker() {
	for {
		time.Sleep(30 * time.Second)
		for _, client := range qm.GetAllClients() {
			if client.Tag == true {
				client.Tag = false
				total_minutes := int(time.Now().Sub(client.RegTime).Minutes())
				hours := total_minutes / 60
				minutes := total_minutes % 60
				client.Serve_time = fmt.Sprintf("%dh%dm", hours, minutes)
				if len(client.Current_work) > 0 {
					client.Idle_time = 0
				} else {
					client.Idle_time += 30
				}
				qm.PutClient(client)
			} else {
				//now client must be gone as tag set to false 30 seconds ago and no heartbeat received thereafter
				logger.Event(event.CLIENT_UNREGISTER, "clientid="+client.Id+";name="+client.Name)

				//requeue unfinished workunits associated with the failed client
				qm.ReQueueWorkunitByClient(client.Id)
				//delete the client from client map
				qm.RemoveClient(client.Id)
				//proxy specific
				Self.SubClients -= 1
				notifySubClients(Self.Id, Self.SubClients)
			}
		}
	}
}

//end of client methods

func (qm *ProxyMgr) EnqueueTasksByJobId(jobid string, tasks []*Task) (err error) {
	return
}

func (qm *ProxyMgr) JobRegister() (jid string, err error) {
	return
}

func (qm *ProxyMgr) GetActiveJobs() map[string]bool {
	return nil
}

func (qm *ProxyMgr) IsJobRegistered(id string) bool {
	return false
}

func (qm *ProxyMgr) GetSuspendJobs() map[string]bool {
	return nil
}

func (qm *ProxyMgr) SuspendJob(jobid string, reason string, id string) (err error) {
	return
}

func (qm *ProxyMgr) ResumeSuspendedJobs() (num int) {
	return
}

func (qm *ProxyMgr) ResumeSuspendedJobsByUser(u *user.User) (num int) {
	return
}

func (qm *ProxyMgr) DeleteJob(jobid string) (err error) {
	return
}

func (qm *ProxyMgr) DeleteJobByUser(jobid string, u *user.User) (err error) {
	return
}

func (qm *ProxyMgr) DeleteSuspendedJobs() (num int) {
	return
}

func (qm *ProxyMgr) DeleteSuspendedJobsByUser(u *user.User) (num int) {
	return
}

func (qm *ProxyMgr) DeleteZombieJobs() (num int) {
	return
}

func (qm *ProxyMgr) DeleteZombieJobsByUser(u *user.User) (num int) {
	return
}

//resubmit a suspended job
func (qm *ProxyMgr) ResumeSuspendedJob(id string) (err error) {
	//Load job by id
	return
}

//resubmit a suspended job if user has rights
func (qm *ProxyMgr) ResumeSuspendedJobByUser(id string, u *user.User) (err error) {
	//Load job by id
	return
}

//re-submit a job in db but not in the queue (caused by server restarting)
func (qm *ProxyMgr) ResubmitJob(id string) (err error) {
	return
}

//recover jobs not completed before awe-server restarts
func (qm *ProxyMgr) RecoverJobs() (err error) {
	return
}

//recompute jobs from specified task stage
func (qm *ProxyMgr) RecomputeJob(jobid string, stage string) (err error) {
	return
}

func (qm *ProxyMgr) UpdateGroup(jobid string, newgroup string) (err error) {
	return
}

func (qm *ProxyMgr) UpdatePriority(jobid string, priority int) (err error) {
	return
}

func (qm *ProxyMgr) FinalizeWorkPerf(string, string) (err error) {
	return
}

func (qm *ProxyMgr) SaveStdLog(string, string, string) (err error) {
	return
}

func (qm *ProxyMgr) GetReportMsg(string, string) (report string, err error) {
	return
}

//---end of job methods
