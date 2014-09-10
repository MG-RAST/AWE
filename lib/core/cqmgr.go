package core

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/AWE/lib/user"
	"github.com/MG-RAST/golib/mgo/bson"
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
				qm.ReQueueWorkunitByClient(clientid)
				//delete the client from client map
				delete(qm.clientMap, clientid)
			}
		}
	}
}

func (qm *CQMgr) ClientHeartBeat(id string, cg *ClientGroup) (hbmsg HBmsg, err error) {
	hbmsg = make(map[string]string, 1)
	if _, ok := qm.clientMap[id]; ok {
		// If the name of the clientgroup (from auth token) does not match the name in the client retrieved, throw an error
		if cg != nil && qm.clientMap[id].Group != cg.Name {
			return nil, errors.New("Clientgroup name in token does not match that in the client.")
		}
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
		if qm.clientMap[id].Status == CLIENT_STAT_DELETED {
			hbmsg["stop"] = id
		}
		//hbmsg["discard"] = strings.Join(workids, ",")
		return hbmsg, nil
	}
	return hbmsg, errors.New(e.ClientNotFound)
}

func (qm *CQMgr) RegisterNewClient(files FormFiles, cg *ClientGroup) (client *Client, err error) {
	if _, ok := files["profile"]; ok {
		client, err = NewProfileClient(files["profile"].Path)
		os.Remove(files["profile"].Path)
	} else {
		client = NewClient()
	}
	if err != nil {
		return nil, err
	}
	// If clientgroup is nil at this point, create a publicly owned clientgroup, with the provided group name (if one doesn't exist with the same name)
	if cg == nil {
		// See if clientgroup already exists with this name
		// If it does and it does not have "public" execution rights, throw error
		// If it doesn't, create one owned by public, and continue with client registration
		// Otherwise proceed with client registration.
		cg, _ = LoadClientGroupByName(client.Group)

		if cg != nil {
			rights := cg.Acl.Check("public")
			if rights["execute"] == false {
				return nil, errors.New("Clientgroup with the group specified by your client exists, but you cannot register with it, without clientgroup token.")
			}
		} else {
			u := &user.User{Uuid: "public"}
			cg, err = CreateClientGroup(client.Group, u)
			if err != nil {
				return nil, err
			}
		}
	}
	// If the name of the clientgroup (from auth token) does not match the name in the client profile, throw an error
	if cg != nil && client.Group != cg.Name {
		return nil, errors.New("Clientgroup name in token does not match that in the client configuration.")
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

func (qm *CQMgr) GetClientByUser(id string, u *user.User) (client *Client, err error) {
	// Get all clientgroups that user owns or that are publicly owned, or all if user is admin
	q := bson.M{}
	clientgroups := new(ClientGroups)
	dbFindClientGroups(q, clientgroups)
	filtered_clientgroups := map[string]bool{}
	for _, cg := range *clientgroups {
		if (u.Uuid != "public" && (cg.Acl.Owner == u.Uuid || u.Admin == true || cg.Acl.Owner == "public")) ||
			(u.Uuid == "public" && conf.CLIENT_AUTH_REQ == false && cg.Acl.Owner == "public") {
			filtered_clientgroups[cg.Name] = true
		}
	}

	if client, ok := qm.clientMap[id]; ok {
		if val, exists := filtered_clientgroups[client.Group]; exists == true && val == true {
			return client, nil
		}
	}
	return nil, errors.New(e.ClientNotFound)
}

func (qm *CQMgr) GetAllClients() []*Client {
	clients := []*Client{}
	for _, client := range qm.clientMap {
		clients = append(clients, client)
	}
	return clients
}

func (qm *CQMgr) GetAllClientsByUser(u *user.User) []*Client {
	// Get all clientgroups that user owns or that are publicly owned, or all if user is admin
	q := bson.M{}
	clientgroups := new(ClientGroups)
	dbFindClientGroups(q, clientgroups)
	filtered_clientgroups := map[string]bool{}
	for _, cg := range *clientgroups {
		if (u.Uuid != "public" && (cg.Acl.Owner == u.Uuid || u.Admin == true || cg.Acl.Owner == "public")) ||
			(u.Uuid == "public" && conf.CLIENT_AUTH_REQ == false && cg.Acl.Owner == "public") {
			filtered_clientgroups[cg.Name] = true
		}
	}

	clients := []*Client{}
	for _, client := range qm.clientMap {
		if val, exists := filtered_clientgroups[client.Group]; exists == true && val == true {
			clients = append(clients, client)
		}
	}
	return clients
}

func (qm *CQMgr) DeleteClient(id string) (err error) {
	if client, ok := qm.clientMap[id]; ok {
		client.Status = CLIENT_STAT_DELETED
		return
	}
	return errors.New("client not found")
}

func (qm *CQMgr) DeleteClientByUser(id string, u *user.User) (err error) {
	// Get all clientgroups that user owns or that are publicly owned, or all if user is admin
	q := bson.M{}
	clientgroups := new(ClientGroups)
	dbFindClientGroups(q, clientgroups)
	filtered_clientgroups := map[string]bool{}
	for _, cg := range *clientgroups {
		if (u.Uuid != "public" && (cg.Acl.Owner == u.Uuid || u.Admin == true || cg.Acl.Owner == "public")) ||
			(u.Uuid == "public" && conf.CLIENT_AUTH_REQ == false && cg.Acl.Owner == "public") {
			filtered_clientgroups[cg.Name] = true
		}
	}

	if client, ok := qm.clientMap[id]; ok {
		if val, exists := filtered_clientgroups[client.Group]; exists == true && val == true {
			client.Status = CLIENT_STAT_DELETED
			return
		}
		return errors.New(e.UnAuth)
	}
	return errors.New("client not found")
}

func (qm *CQMgr) SuspendClient(id string) (err error) {
	if client, ok := qm.clientMap[id]; ok {
		if client.Status == CLIENT_STAT_ACTIVE_IDLE || client.Status == CLIENT_STAT_ACTIVE_BUSY {
			client.Status = CLIENT_STAT_SUSPEND
			qm.ReQueueWorkunitByClient(id)
			return
		}
		return errors.New("client is not in active state")
	}
	return errors.New("client not found")
}

func (qm *CQMgr) SuspendClientByUser(id string, u *user.User) (err error) {
	// Get all clientgroups that user owns or that are publicly owned, or all if user is admin
	q := bson.M{}
	clientgroups := new(ClientGroups)
	dbFindClientGroups(q, clientgroups)
	filtered_clientgroups := map[string]bool{}
	for _, cg := range *clientgroups {
		if (u.Uuid != "public" && (cg.Acl.Owner == u.Uuid || u.Admin == true || cg.Acl.Owner == "public")) ||
			(u.Uuid == "public" && conf.CLIENT_AUTH_REQ == false && cg.Acl.Owner == "public") {
			filtered_clientgroups[cg.Name] = true
		}
	}

	if client, ok := qm.clientMap[id]; ok {
		if val, exists := filtered_clientgroups[client.Group]; exists == true && val == true {
			if client.Status == CLIENT_STAT_ACTIVE_IDLE || client.Status == CLIENT_STAT_ACTIVE_BUSY {
				client.Status = CLIENT_STAT_SUSPEND
				qm.ReQueueWorkunitByClient(id)
				return
			}
			return errors.New("client is not in active state")
		}
		return errors.New(e.UnAuth)
	}
	return errors.New("client not found")
}

func (qm *CQMgr) ResumeClient(id string) (err error) {
	if client, ok := qm.clientMap[id]; ok {
		if client.Status == CLIENT_STAT_SUSPEND {
			client.Status = CLIENT_STAT_ACTIVE_IDLE
			return
		}
		return errors.New("client is not in suspended state")
	}
	return errors.New("client not found")
}

func (qm *CQMgr) ResumeClientByUser(id string, u *user.User) (err error) {
	// Get all clientgroups that user owns or that are publicly owned, or all if user is admin
	q := bson.M{}
	clientgroups := new(ClientGroups)
	dbFindClientGroups(q, clientgroups)
	filtered_clientgroups := map[string]bool{}
	for _, cg := range *clientgroups {
		if (u.Uuid != "public" && (cg.Acl.Owner == u.Uuid || u.Admin == true || cg.Acl.Owner == "public")) ||
			(u.Uuid == "public" && conf.CLIENT_AUTH_REQ == false && cg.Acl.Owner == "public") {
			filtered_clientgroups[cg.Name] = true
		}
	}

	if client, ok := qm.clientMap[id]; ok {
		if val, exists := filtered_clientgroups[client.Group]; exists == true && val == true {
			if client.Status == CLIENT_STAT_SUSPEND {
				client.Status = CLIENT_STAT_ACTIVE_IDLE
				return
			}
			return errors.New("client is not in suspended state")
		}
		return errors.New(e.UnAuth)
	}
	return errors.New("client not found")
}

func (qm *CQMgr) ResumeSuspendedClients() (count int) {
	for _, client := range qm.clientMap {
		if client.Status == CLIENT_STAT_SUSPEND {
			client.Status = CLIENT_STAT_ACTIVE_IDLE
			count += 1
		}
	}
	return count
}

func (qm *CQMgr) ResumeSuspendedClientsByUser(u *user.User) (count int) {
	// Get all clientgroups that user owns or that are publicly owned, or all if user is admin
	q := bson.M{}
	clientgroups := new(ClientGroups)
	dbFindClientGroups(q, clientgroups)
	filtered_clientgroups := map[string]bool{}
	for _, cg := range *clientgroups {
		if (u.Uuid != "public" && (cg.Acl.Owner == u.Uuid || u.Admin == true || cg.Acl.Owner == "public")) ||
			(u.Uuid == "public" && conf.CLIENT_AUTH_REQ == false && cg.Acl.Owner == "public") {
			filtered_clientgroups[cg.Name] = true
		}
	}

	for _, client := range qm.clientMap {
		if val, exists := filtered_clientgroups[client.Group]; exists == true && val == true && client.Status == CLIENT_STAT_SUSPEND {
			client.Status = CLIENT_STAT_ACTIVE_IDLE
			count += 1
		}
	}
	return count
}

func (qm *CQMgr) SuspendAllClients() (count int) {
	for _, client := range qm.clientMap {
		if client.Status == CLIENT_STAT_ACTIVE_IDLE || client.Status == CLIENT_STAT_ACTIVE_BUSY {
			qm.SuspendClient(client.Id)
			count += 1
		}
	}
	return count
}

func (qm *CQMgr) SuspendAllClientsByUser(u *user.User) (count int) {
	// Get all clientgroups that user owns or that are publicly owned, or all if user is admin
	q := bson.M{}
	clientgroups := new(ClientGroups)
	dbFindClientGroups(q, clientgroups)
	filtered_clientgroups := map[string]bool{}
	for _, cg := range *clientgroups {
		if (u.Uuid != "public" && (cg.Acl.Owner == u.Uuid || u.Admin == true || cg.Acl.Owner == "public")) ||
			(u.Uuid == "public" && conf.CLIENT_AUTH_REQ == false && cg.Acl.Owner == "public") {
			filtered_clientgroups[cg.Name] = true
		}
	}

	for _, client := range qm.clientMap {
		if val, exists := filtered_clientgroups[client.Group]; exists == true && val == true && (client.Status == CLIENT_STAT_ACTIVE_IDLE || client.Status == CLIENT_STAT_ACTIVE_BUSY) {
			qm.SuspendClient(client.Id)
			count += 1
		}
	}
	return count
}

func (qm *CQMgr) UpdateSubClients(id string, count int) {
	if _, ok := qm.clientMap[id]; ok {
		qm.clientMap[id].SubClients = count
	}
}

func (qm *CQMgr) UpdateSubClientsByUser(id string, count int, u *user.User) {
	// Get all clientgroups that user owns or that are publicly owned, or all if user is admin
	q := bson.M{}
	clientgroups := new(ClientGroups)
	dbFindClientGroups(q, clientgroups)
	filtered_clientgroups := map[string]bool{}
	for _, cg := range *clientgroups {
		if (u.Uuid != "public" && (cg.Acl.Owner == u.Uuid || u.Admin == true || cg.Acl.Owner == "public")) ||
			(u.Uuid == "public" && conf.CLIENT_AUTH_REQ == false && cg.Acl.Owner == "public") {
			filtered_clientgroups[cg.Name] = true
		}
	}

	if client, ok := qm.clientMap[id]; ok {
		if val, exists := filtered_clientgroups[client.Group]; exists == true && val == true {
			qm.clientMap[id].SubClients = count
		}
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
	if qm.clientMap[client_id].Status == CLIENT_STAT_DELETED {
		delete(qm.clientMap, client_id)
		return nil, errors.New(e.ClientDeleted)
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
		if qm.clientMap[client_id].Status == CLIENT_STAT_ACTIVE_IDLE {
			qm.clientMap[client_id].Status = CLIENT_STAT_ACTIVE_BUSY
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
	logger.Debug(2, fmt.Sprintf("popWorks filtered: %d (0 meansNoEligibleWorkunitFound)", filtered))
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
			logger.Error(fmt.Sprintf("error: workunit %s is in wait queue but not in workMap", id))
			continue
		}
		work := qm.workQueue.workMap[id]

		// In case of edge case where pointer to workunit is in queue but workunit has been deleted
		// If work.Info is nil, this will cause errors in execution
		// These will be deleted by servermgr.updateQueue()
		if work == nil || work.Info == nil {
			continue
		}

		//skip works that are in the client's skip-list
		if contains(client.Skip_work, work.Id) {
			logger.Debug(2, fmt.Sprintf("2) contains(client.Skip_work, work.Id) %s", id))
			continue
		}
		//skip works that have dedicate client groups which this client doesn't belong to
		if len(work.Info.ClientGroups) > 0 {
			eligible_groups := strings.Split(work.Info.ClientGroups, ",")
			if !contains(eligible_groups, client.Group) {
				logger.Debug(2, fmt.Sprintf("3) !contains(eligible_groups, client.Group) %s", id))
				continue
			}
		}
		//append works whos apps are supported by the client
		if contains(client.Apps, work.Cmd.Name) || contains(client.Apps, conf.ALL_APP) {
			ids = append(ids, id)
		} else {
			logger.Debug(2, fmt.Sprintf("3) contains(client.Apps, work.Cmd.Name) || contains(client.Apps, conf.ALL_APP) %s", id))
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

func (qm *CQMgr) ShowWorkunitsByUser(status string, u *user.User) (workunits []*Workunit) {
	// Only returns workunits of jobs that the user has read access to or is the owner of.  If user is admin, return all.
	for _, work := range qm.workQueue.workMap {
		if jobid, err := GetJobIdByWorkId(work.Id); err == nil {
			if job, err := LoadJob(jobid); err == nil {
				rights := job.Acl.Check(u.Uuid)
				if job.Acl.Owner == u.Uuid || rights["read"] == true || u.Admin == true {
					if work.State == status || status == "" {
						workunits = append(workunits, work)
					}
				}
			}
		}
	}
	return workunits
}

func (qm *CQMgr) EnqueueWorkunit(work *Workunit) (err error) {
	err = qm.workQueue.Add(work)
	return
}

func (qm *CQMgr) ReQueueWorkunitByClient(clientid string) (err error) {
	workids := qm.getWorkByClient(clientid)
	for _, workid := range workids {
		if qm.workQueue.Has(workid) {
			jobid, _ := GetJobIdByWorkId(workid)
			if job, err := LoadJob(jobid); err == nil {
				if contains(JOB_STATS_ACTIVE, job.State) { //only requeue workunits belonging to active jobs (rule out suspended jobs)
					qm.workQueue.StatusChange(workid, WORK_STAT_QUEUED)
					logger.Event(event.WORK_REQUEUE, "workid="+workid)
				}
			}
		}
	}
	return
}

//---end of workunit methods
