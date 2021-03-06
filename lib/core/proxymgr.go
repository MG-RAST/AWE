package core

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/MG-RAST/AWE/lib/conf"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/AWE/lib/user"
)

type ProxyMgr struct {
	CQMgr
}

func (qm *ProxyMgr) Lock()    {}
func (qm *ProxyMgr) Unlock()  {}
func (qm *ProxyMgr) RLock()   {}
func (qm *ProxyMgr) RUnlock() {}

func (qm *ProxyMgr) UpdateQueueLoop() {}

func NewProxyMgr() *ProxyMgr {
	return &ProxyMgr{
		CQMgr: CQMgr{
			clientMap:    *NewClientMap(),
			workQueue:    NewWorkQueue(),
			suspendQueue: false,
			coReq:        make(chan CheckoutRequest),
			feedback:     make(chan Notice),
			coSem:        make(chan int, 1), //non-blocking buffered channel
		},
	}
}

func (qm *ProxyMgr) ClientHandle() {
	for {
		select {
		case coReq := <-qm.coReq:
			logger.Debug(2, fmt.Sprintf("proxymgr: workunit checkout request received, Req=%v\n", coReq))
			var ack CoAck
			if qm.suspendQueue {
				// queue is suspended, return suspend error
				ack = CoAck{workunits: nil, err: errors.New(e.QueueSuspend)}
			} else {
				works, err := qm.popWorks(coReq)
				ack = CoAck{workunits: works, err: err}
			}
			//qm.coAck <- ack
			coReq.response <- ack
		case notice := <-qm.feedback:
			id_str, _ := notice.ID.String()
			logger.Debug(2, "proxymgr: workunit feedback received, workid=%s, status=%s, clientid=%s\n", id_str, notice.Status, notice.WorkerID)
			if err := qm.handleNoticeWorkDelivered(&notice); err != nil {
				logger.Error("handleNoticeWorkDelivered(): " + err.Error())
			}
		}
	}
}

func (qm *ProxyMgr) NoticeHandle() {
	//TODO copy code from ClientHandle, and/or reuse server code
	return
}

func (qm *ProxyMgr) SuspendQueue() {
	return
}

func (qm *ProxyMgr) ResumeQueue() {
	return
}

func (qm *ProxyMgr) QueueStatus() string {
	return ""
}

func (qm *ProxyMgr) GetQueue(name string) interface{} {
	return nil
}

func (qm *ProxyMgr) GetJsonStatus() (status map[string]map[string]int, err error) {
	return status, err
}

func (qm *ProxyMgr) GetTextStatus() string {
	return ""
}

//---end of mgr methods

// workunit methods

//handle feedback from a client about the execution of a workunit
func (qm *ProxyMgr) handleNoticeWorkDelivered(notice *Notice) (err error) {
	//relay the notice to the server
	perf := new(WorkPerf)
	workid := notice.ID
	clientid := notice.WorkerID
	client, ok, err := qm.GetClient(clientid, true)
	if err != nil {
		return
	}
	if ok {
		//delete(client.Current_work, workid)
		client.LockNamed("ProxyMgr/handleNoticeWorkDelivered A2")
		err = client.AssignedWork.Delete(notice.ID, false)
		if err != nil {
			return
		}

		qm.AddClient(client, true)
		client.Unlock()
	}
	work, ok, err := qm.workQueue.Get(workid)
	if err != nil {
		return
	}
	if ok {
		err = work.SetState(notice.Status, "")
		if err != nil {
			return
		}
		if err = proxy_relay_workunit(work, perf); err != nil {
			return
		}
		if work.State == WORK_STAT_DONE {
			client, ok, xerr := qm.GetClient(clientid, true)
			if xerr != nil {
				err = xerr
				return
			}
			if ok {
				client.IncrementTotalCompleted()
				client.LastFailed = 0 //reset last consecutive failures
				qm.AddClient(client, true)
			}
		} else if work.State == WORK_STAT_ERROR {
			client, ok, xerr := qm.GetClient(clientid, true)
			if xerr != nil {
				err = xerr
				return
			}
			if ok {
				client.LockNamed("ProxyMgr/handleNoticeWorkDelivered B")
				err = client.AppendSkipwork(workid, false)
				if err != nil {
					return
				}
				err = client.IncrementTotalFailed(false)
				if err != nil {
					return
				}
				client.LastFailed += 1 //last consecutive failures
				if conf.MAX_CLIENT_FAILURE != 0 {
					if client.LastFailed >= conf.MAX_CLIENT_FAILURE {
						client.Suspend("MAX_CLIENT_FAILURE reached", false)
					}
				}
				qm.AddClient(client, false)
				client.Unlock()
			}
		}
		//qm.workQueue.Put(work)
	}
	return
}

func (qm *ProxyMgr) FetchDataToken(workunit *Workunit, clientid string) (token string, err error) {
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
		if err != nil {
			return
		}
	} else {
		client = NewClient()
	}

	client.LockNamed("proxymgr/RegisterNewClient")
	defer client.Unlock()

	// If the name of the clientgroup does not match the name in the client profile, throw an error
	if cg != nil && client.Group != cg.Name {
		return nil, errors.New("Clientgroup name in token does not match that in the client configuration.")
	}
	qm.AddClient(client, true)
	cw_length, err := client.CurrentWork.Length(false)
	if err != nil {
		return
	}
	if cw_length > 0 { //re-registered client
		// move already checked-out workunit from waiting queue (workMap) to checked-out list (coWorkMap)

		work_list, _ := client.CurrentWork.Get_list(false)

		for _, workid := range work_list {
			has_work, err := qm.workQueue.Has(workid)
			if err != nil {
				continue
			}
			if has_work {
				qm.workQueue.StatusChange(workid, nil, WORK_STAT_CHECKOUT, "")
			}
		}

	}
	//proxy specific
	Self.SubClients += 1
	_ = notifySubClients(Self.ID, Self.SubClients)
	return
}

func (qm *ProxyMgr) ClientChecker() {
	for {
		time.Sleep(30 * time.Second)

		delete_clients := []string{}

		clients, err := qm.clientMap.GetClients()
		if err != nil {
			logger.Error("ProxyMgr/ClientChecker: %s", err.Error())
			continue
		}
		for _, client := range clients {
			//for _, client := range qm.GetAllClients() {
			client.LockNamed("ProxyMgr/ClientChecker")
			if client.Tag == true {
				client.Tag = false
				total_minutes := int(time.Now().Sub(client.RegTime).Minutes())
				hours := total_minutes / 60
				minutes := total_minutes % 60
				client.ServeTime = fmt.Sprintf("%dh%dm", hours, minutes)

				//if cw_length > 0 {
				//	client.Idle_time = 0
				//} else {
				//	client.Idle_time += 30
				//}

			} else {
				//now client must be gone as tag set to false 30 seconds ago and no heartbeat received thereafter
				logger.Event(event.CLIENT_UNREGISTER, "clientid="+client.ID)

				//qm.RemoveClient(client.Id)
				delete_clients = append(delete_clients, client.ID)

			}
			client.Unlock()
		}

		// Now delete clients
		if len(delete_clients) > 0 {
			//qm.clientMap.LockNamed("ClientChecker")
			for _, client_id := range delete_clients {

				//client, ok := qm.workQueue.workMap.Get(client_id)
				client, ok, err := qm.clientMap.Get(client_id, true)
				if err != nil {
					continue
				}
				if ok {
					//requeue unfinished workunits associated with the failed client
					qm.ReQueueWorkunitByClient(client, true)
					//delete the client from client map

					qm.RemoveClient(client_id, true)

					//proxy specific
					Self.SubClients -= 1
					notifySubClients(Self.ID, Self.SubClients)
				}
			}
			//qm.clientMap.Unlock()
		}
	}
}

//end of client methods

func (qm *ProxyMgr) EnqueueTasksByJobId(jobid string) (err error) {
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

func (qm *ProxyMgr) SuspendJob(jobid string, jerror *JobError) (err error) {
	return
}

func (qm *ProxyMgr) ResumeSuspendedJobsByUser(u *user.User) (num int) {
	return
}

func (qm *ProxyMgr) DeleteJobByUser(jobid string, u *user.User, full bool) (err error) {
	return
}

func (qm *ProxyMgr) DeleteSuspendedJobsByUser(u *user.User, full bool) (num int) {
	return
}

func (qm *ProxyMgr) DeleteZombieJobsByUser(u *user.User, full bool) (num int) {
	return
}

//resubmit a suspended job if user has rights
func (qm *ProxyMgr) ResumeSuspendedJobByUser(id string, u *user.User) (recovered bool, err error) {
	//Load job by id
	return
}

//re-submit a job in db but not in the queue (caused by server restarting)
func (qm *ProxyMgr) ResubmitJob(id string) (err error) {
	return
}

//recover job not in queue
func (qm *ProxyMgr) RecoverJob(id string, job *Job) (err error) {
	return
}

//recover jobs not completed before awe-server restarts
func (qm *ProxyMgr) RecoverJobs() (recovered int, total int, err error) {
	return
}

//recompute jobs from specified task stage
func (qm *ProxyMgr) RecomputeJob(jobid string, stage string) (err error) {
	return
}

func (qm *ProxyMgr) UpdateQueueToken(job *Job) (err error) {
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
