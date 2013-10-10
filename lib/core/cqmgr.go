package core

import (
	"errors"
	"fmt"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"os"
	"strings"
	"time"
)

type CQMgr struct {
	clientMap map[string]*Client
	workQueue *WQueue
	reminder  chan bool
	coReq     chan CoReq  //workunit checkout request (WorkController -> qmgr.Handler)
	coAck     chan CoAck  //workunit checkout item including data and err (qmgr.Handler -> WorkController)
	feedback  chan Notice //workunit execution feedback (WorkController -> qmgr.Handler)
	coSem     chan int    //semaphore for checkout (mutual exclusion between different clients)
}

func NewCQMgr() *CQMgr {
	return &CQMgr{
		clientMap: map[string]*Client{},
		workQueue: NewWQueue(),
		reminder:  make(chan bool),
		coReq:     make(chan CoReq),
		coAck:     make(chan CoAck),
		feedback:  make(chan Notice),
		coSem:     make(chan int, 1), //non-blocking buffered channel
	}
}

//to-do: consider separate some independent tasks into another goroutine to handle

func (qm *CQMgr) Handle() {
	for {
		select {
		case coReq := <-qm.coReq:
			logger.Debug(2, fmt.Sprintf("qmgr: workunit checkout request received, Req=%v\n", coReq))
			works, err := qm.popWorks(coReq)
			ack := CoAck{workunits: works, err: err}
			qm.coAck <- ack

		case notice := <-qm.feedback:
			logger.Debug(2, fmt.Sprintf("qmgr: workunit feedback received, workid=%s, status=%s, clientid=%s\n", notice.WorkId, notice.Status, notice.ClientId))
			if err := qm.handleWorkStatusChange(notice); err != nil {
				logger.Error("handleWorkStatusChange(): " + err.Error())
			}

		case <-qm.reminder:
			logger.Debug(3, "time to update workunit queue....\n")
		}
	}
}

func (qm *CQMgr) Timer() {
	for {
		time.Sleep(10 * time.Second)
		qm.reminder <- true
	}
}

// show functions used in debug
func (qm *CQMgr) ShowWorkQueue() {
	fmt.Printf("current queuing workunits (%d):\n", qm.workQueue.Len())
	for key, _ := range qm.workQueue.workMap {
		fmt.Printf("workunit id: %s\n", key)
	}
	return
}

//---end of mgr methods

//--------client methods-------

func (qm *CQMgr) ClientChecker() {
	for {
		time.Sleep(30 * time.Second)
		for clientid, client := range qm.clientMap {
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
			} else {
				//now client must be gone as tag set to false 30 seconds ago and no heartbeat received thereafter
				logger.Event(event.CLIENT_UNREGISTER, "clientid="+clientid+";name="+qm.clientMap[clientid].Name)

				//requeue unfinished workunits associated with the failed client
				workids := qm.getWorkByClient(clientid)
				for _, workid := range workids {
					if qm.workQueue.Has(workid) {
						qm.workQueue.StatusChange(workid, WORK_STAT_QUEUED)
						logger.Event(event.WORK_REQUEUE, "workid="+workid)
					}
				}
				//delete the client from client map
				delete(qm.clientMap, clientid)
			}
		}
	}
}

func (qm *CQMgr) ClientHeartBeat(id string) (hbmsg HBmsg, err error) {
	hbmsg = make(map[string]string, 1)
	if _, ok := qm.clientMap[id]; ok {
		qm.clientMap[id].Tag = true
		logger.Debug(3, "HeartBeatFrom:"+"clientid="+id+",name="+qm.clientMap[id].Name)

		//get suspended workunit that need the client to discard
		workids := qm.getWorkByClient(id)
		suspended := []string{}
		for _, workid := range workids {
			if work, ok := qm.workQueue.workMap[workid]; ok {
				if work.State == WORK_STAT_SUSPEND {
					suspended = append(suspended, workid)
				}
			}
		}
		if len(suspended) > 0 {
			hbmsg["discard"] = strings.Join(suspended, ",")
		}
		//hbmsg["discard"] = strings.Join(workids, ",")
		return hbmsg, nil
	}
	return hbmsg, errors.New(e.ClientNotFound)
}

func (qm *CQMgr) RegisterNewClient(files FormFiles) (client *Client, err error) {
	if _, ok := files["profile"]; ok {
		client, err = NewProfileClient(files["profile"].Path)
		os.Remove(files["profile"].Path)
	} else {
		client = NewClient()
	}
	if err != nil {
		return nil, err
	}
	qm.clientMap[client.Id] = client

	if len(client.Current_work) > 0 { //re-registered client
		// move already checked-out workunit from waiting queue (workMap) to checked-out list (coWorkMap)
		for workid, _ := range client.Current_work {
			if qm.workQueue.Has(workid) {
				qm.workQueue.StatusChange(workid, WORK_STAT_CHECKOUT)
			}
		}
	}
	return
}

func (qm *CQMgr) GetClient(id string) (client *Client, err error) {
	if client, ok := qm.clientMap[id]; ok {
		return client, nil
	}
	return nil, errors.New(e.ClientNotFound)
}

func (qm *CQMgr) GetAllClients() []*Client {
	var clients []*Client
	for _, client := range qm.clientMap {
		clients = append(clients, client)
	}
	return clients
}

func (qm *CQMgr) DeleteClient(id string) {
	delete(qm.clientMap, id)
}

func (qm *CQMgr) UpdateSubClients(id string, count int) {
	if _, ok := qm.clientMap[id]; ok {
		qm.clientMap[id].SubClients = count
	}
}

//--------end client methods-------

//-------start of workunit methods---

func (qm *CQMgr) CheckoutWorkunits(req_policy string, client_id string, num int) (workunits []*Workunit, err error) {
	//precheck if the client is registered
	if _, hasClient := qm.clientMap[client_id]; !hasClient {
		return nil, errors.New(e.ClientNotFound)
	}
	if qm.clientMap[client_id].Status == CLIENT_STAT_SUSPEND {
		return nil, errors.New(e.ClientSuspended)
	}

	//lock semephore, at one time only one client's checkout request can be served
	qm.coSem <- 1

	req := CoReq{policy: req_policy, fromclient: client_id, count: num}
	qm.coReq <- req
	ack := <-qm.coAck

	if ack.err == nil {
		for _, work := range ack.workunits {
			qm.clientMap[client_id].Total_checkout += 1
			qm.clientMap[client_id].Current_work[work.Id] = true
		}
	}

	//unlock
	<-qm.coSem

	return ack.workunits, ack.err
}

func (qm *CQMgr) GetWorkById(id string) (workunit *Workunit, err error) {
	if workunit, hasid := qm.workQueue.workMap[id]; hasid {
		return workunit, nil
	}
	return nil, errors.New(fmt.Sprintf("no workunit found with id %s", id))
}

func (qm *CQMgr) NotifyWorkStatus(notice Notice) {
	qm.feedback <- notice
	return
}

func (qm *CQMgr) popWorks(req CoReq) (works []*Workunit, err error) {
	filtered := qm.filterWorkByClient(req.fromclient)
	if len(filtered) == 0 {
		return nil, errors.New(e.NoEligibleWorkunitFound)
	}
	works, err = qm.workQueue.selectWorkunits(filtered, req.policy, req.count)
	if err == nil { //get workunits successfully, put them into coWorkMap
		for _, work := range works {
			if _, ok := qm.workQueue.workMap[work.Id]; ok {
				qm.workQueue.workMap[work.Id].Client = req.fromclient
				qm.workQueue.workMap[work.Id].CheckoutTime = time.Now()
			}
			qm.workQueue.StatusChange(work.Id, WORK_STAT_CHECKOUT)
		}
	}
	return
}

func (qm *CQMgr) filterWorkByClient(clientid string) (ids []string) {
	client := qm.clientMap[clientid]
	for id, _ := range qm.workQueue.wait {
		if _, ok := qm.workQueue.workMap[id]; !ok {
			continue
		}
		work := qm.workQueue.workMap[id]
		//skip works that are in the client's skip-list
		if contains(client.Skip_work, work.Id) {
			continue
		}
		//skip works that have dedicate client groups which this client doesn't belong to
		if len(work.Info.ClientGroups) > 0 {
			eligible_groups := strings.Split(work.Info.ClientGroups, ",")
			if !contains(eligible_groups, client.Group) {
				continue
			}
		}
		//append works whos apps are supported by the client
		if contains(client.Apps, work.Cmd.Name) {
			ids = append(ids, id)
		}
	}
	return ids
}

func (qm *CQMgr) getWorkByClient(clientid string) (ids []string) {
	if client, ok := qm.clientMap[clientid]; ok {
		for id, _ := range client.Current_work {
			ids = append(ids, id)
		}
	}
	return
}

//handle feedback from a client about the execution of a workunit
func (qm *CQMgr) handleWorkStatusChange(notice Notice) (err error) {
	//to be implemented for proxy or server
	return
}

func (qm *CQMgr) FetchDataToken(workid string, clientid string) (token string, err error) {
	//to be implemented for proxy or server
	return
}

func (qm *CQMgr) ShowWorkunits(status string) (workunits []*Workunit) {
	for _, work := range qm.workQueue.workMap {
		if work.State == status || status == "" {
			workunits = append(workunits, work)
		}
	}
	return workunits
}

func (qm *CQMgr) EnqueueWorkunit(work *Workunit) (err error) {
	err = qm.workQueue.Add(work)
	return
}

//---end of workunit methods
